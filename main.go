package main

import (
	"fmt"
	"gateway-knx/lib"
	"time"

	eria "github.com/project-eria/eria-core"
	zlog "github.com/rs/zerolog/log"
	"github.com/vapourismo/knx-go/knx"
	"github.com/vapourismo/knx-go/knx/cemi"
)

var config = struct {
	Host        string             `yaml:"host"`
	Port        uint               `yaml:"port" default:"80"`
	ExposedAddr string             `yaml:"exposedAddr"`
	Gateway     configGateway      `yaml:"gateway"`
	TunnelMode  bool               `yaml:"tunnelMode" default:"false"`
	Devices     []lib.ConfigDevice `yaml:"devices"`
}{}

type configGateway struct {
	Host string `yaml:"host" default:"127.0.0.1"`
	Port int    `yaml:"port" default:"3671"`
}

// var _devs map[string]*device.Device

func init() {
	eria.Init("ERIA KNX Gateway")
}

func main() {
	defer func() {
		zlog.Info().Msg("[main] Stopped")
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

	lib.ConnectKNX(GWAddr, knxConfig)

	// Close upon exiting. Even if the gateway closes the connection, we still have to clean up.
	defer lib.CloseKNX()

	eriaServer := eria.NewServer(config.Host, config.Port, config.ExposedAddr, "")

	setupThings(eriaServer)

	// Receive messages from the gateway
	go lib.UpdateFromKNX()

	eriaServer.StartServer()
}

// setup : create devices, register ...
func setupThings(eriaServer *eria.EriaServer) {
	// var addresses []string
	for i := range config.Devices {
		confDev := &config.Devices[i]

		for _, conf := range confDev.States {
			conf := conf
			group, err := cemi.NewGroupAddrString(conf.GrpAddr)
			if err != nil {
				zlog.Warn().Err(err).Msg("[main]")
				break
			}
			conf.GroupRead = &group
		}
		for _, conf := range confDev.Actions {
			conf := conf
			group, err := cemi.NewGroupAddrString(conf.GrpAddr)
			if err != nil {
				zlog.Warn().Err(err).Msg("[main]")
				break
			}
			conf.GroupWrite = &group
		}

		td, _ := eria.NewThingDescription(
			"eria:gateway:knx:"+confDev.Ref,
			eria.AppVersion,
			confDev.Ref,
			confDev.Name,
			[]string{confDev.Type},
		)

		eriaThing, _ := eriaServer.AddThing(confDev.Ref, td)

		_, err := lib.NewKNXThing(confDev, eriaThing)
		if err != nil {
			zlog.Error().Str("device", confDev.Ref).Err(err).Msg("[main]")
			continue
		}
	}
}
