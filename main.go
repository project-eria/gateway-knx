package main

import (
	"fmt"
	"time"

	eria "github.com/project-eria/eria-core"
	"github.com/rs/zerolog/log"
	"github.com/vapourismo/knx-go/knx"
	"github.com/vapourismo/knx-go/knx/cemi"
)

var config = struct {
	Host        string         `yaml:"host"`
	Port        uint           `yaml:"port" default:"80"`
	ExposedAddr string         `yaml:"exposedAddr"`
	Gateway     configGateway  `yaml:"gateway"`
	TunnelMode  bool           `yaml:"tunnelMode" default:"false"`
	Devices     []configDevice `yaml:"devices"`
}{}

type configGateway struct {
	Host string `yaml:"host" default:"127.0.0.1"`
	Port int    `yaml:"port" default:"3671"`
}

type configDevice struct {
	Capabilities []string                       `yaml:"capabilities"`
	Name         string                         `yaml:"name"`
	Ref          string                         `yaml:"ref"`
	States       map[string]*configStatesGroup  `yaml:"states"`
	Actions      map[string]*configActionsGroup `yaml:"actions"`
}

type configStatesGroup struct {
	InvertValue bool   `yaml:"invertValue" default:"false"`
	GrpAddr     string `yaml:"grpAddr"`
	handler     func([]byte, bool) error
}

type configActionsGroup struct {
	InvertValue bool   `yaml:"invertValue" default:"false"`
	GrpAddr     string `yaml:"grpAddr"`
	groupWrite  *cemi.GroupAddr
}

// var _devs map[string]*device.Device

// For direct access
var _groupByKNXState map[string]*configStatesGroup

var client knx.GroupTunnel

func init() {
	eria.Init("ERIA KNX Gateway")
}

func main() {
	defer func() {
		log.Info().Msg("[main] Stopped")
	}()

	// Loading config
	eria.LoadConfig(&config)

	// Connect to the KNX gateway.
	GWAddr := fmt.Sprintf("%s:%d", config.Gateway.Host, config.Gateway.Port)
	knxConfig := knx.TunnelConfig{
		ResendInterval:    60 * time.Second, // Wait for one minute
		HeartbeatInterval: 10 * time.Second,
		ResponseTimeout:   9999 * time.Hour, // Don't stop
	}

	var err error
	client, err = knx.NewGroupTunnel(GWAddr, knxConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("[main] Can connect KNX IP Gateway")
	}

	// Close upon exiting. Even if the gateway closes the connection, we still have to clean up.
	defer client.Close()

	eriaServer := eria.NewServer(config.Host, config.Port, config.ExposedAddr, "")

	setupThings(eriaServer)

	// Receive messages from the gateway
	go updateFromKNX()

	eriaServer.StartServer()
}

// setup : create devices, register ...
func setupThings(eriaServer *eria.EriaServer) {
	_groupByKNXState = map[string]*configStatesGroup{}

	// var addresses []string
	for i := range config.Devices {
		confDev := &config.Devices[i]

		td, _ := eria.NewThingDescription(
			"eria:gateway:knx:"+confDev.Ref,
			eria.AppVersion,
			confDev.Ref,
			confDev.Name,
			confDev.Capabilities,
		)

		eriaThing, _ := eriaServer.AddThing(confDev.Ref, td)

		_, err := newKNXThing(confDev, eriaThing)
		if err != nil {
			log.Error().Str("device", confDev.Ref).Err(err).Msg("[main]")
			continue
		}

		for _, conf := range confDev.Actions {
			conf := conf
			group, err := cemi.NewGroupAddrString(conf.GrpAddr)
			if err != nil {
				log.Warn().Err(err).Msg("[main]")
				break
			}
			conf.groupWrite = &group
		}

	}
}

func updateFromKNX() {
	// The inbound channel is closed with the connection.
	for msg := range client.Inbound() {
		addrKNX := msg.Destination.String()
		log.Trace().Str("addrKNX", addrKNX).Msg("[main] Received KNX message from")
		if confGroup, in := _groupByKNXState[addrKNX]; in {
			log.Trace().Str("group", addrKNX).Msg("[main] KNX State group found, process notification")
			if err := confGroup.handler(msg.Data, confGroup.InvertValue); err != nil {
				log.Error().Err(err).Msg("[main]")
			}
		} else {
			log.Trace().Str("group", addrKNX).Msg("[main] KNX State group not in config, ignoring")
		}
	}
}
