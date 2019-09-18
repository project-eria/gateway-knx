package main

import (
	"fmt"
	"os"
	"time"

	"github.com/project-eria/xaal-go"
	"github.com/project-eria/xaal-go/device"
	"github.com/project-eria/xaal-go/message"
	"github.com/project-eria/xaal-go/schemas"
	"github.com/project-eria/xaal-go/utils"

	"github.com/project-eria/eria-base"

	logger "github.com/project-eria/eria-logger"
	"github.com/vapourismo/knx-go/knx/cemi"

	"github.com/vapourismo/knx-go/knx"
)

var (
	// Version is a placeholder that will receive the git tag version during build time
	Version = "-"
)

const configFile = "gateway-knx.json"

func setupDev(dev *device.Device, info string) {
	dev.VendorID = "ERIA"
	dev.ProductID = "KNXIP"
	dev.Info = info
	dev.Version = Version
}

var config = struct {
	GWXaalAddr  string
	GatewayIP   string `default:"127.0.0.1"`
	GatewayPort int    `default:"3671"`
	TunnelMode  bool   `default:"false"`
	Devices     []configDevice
}{}

type configDevice struct {
	Type     string
	Name     string
	XaalAddr string
	Groups   []configGroup
}

type configGroup struct {
	Attribute    string
	InvertValue  bool `default:"false"`
	GrpAddrState string
	GrpAddrWrite string
	groupWrite   *cemi.GroupAddr
	device       *configDevice
}

var _devs map[string]*device.Device

// For direct access
var _configByXAAL map[string]map[string]*configGroup
var _configByKNXWrite map[string]*configGroup
var _configByKNXState map[string]*configGroup

var client knx.GroupTunnel

func main() {
	defer os.Exit(0)

	eria.AddShowVersion(Version)

	logger.Module("main").Infof("Starting Gateway KNX %s...", Version)

	// Loading config
	cm := eria.LoadConfig(configFile, &config)

	defer cm.Close()

	// Init xAAL engine
	eria.InitEngine()

	setup()

	// Save for new Address during setup
	cm.Save()

	xaal.AddRxHandler(updateFromXAAL)

	// Connect to the KNX gateway.
	GWAddr := fmt.Sprintf("%s:%d", config.GatewayIP, config.GatewayPort)
	knxConfig := knx.TunnelConfig{
		ResendInterval:    60 * time.Second, // Wait for one minute
		HeartbeatInterval: 10 * time.Second,
		ResponseTimeout:   9999 * time.Hour, // Don't stop
	}

	var err error
	client, err = knx.NewGroupTunnel(GWAddr, knxConfig)
	if err != nil {
		logger.Module("main").WithError(err).Fatal("Can connect KNX IP Gateway")
	}

	// Close upon exiting. Even if the gateway closes the connection, we still have to clean up.
	defer client.Close()

	// Launch the xAAL engine
	go xaal.Run()
	defer xaal.Stop()

	// Receive messages from the gateway
	go updateFromKNX()

	eria.WaitForExit()
}

// setup : create devices, register ...
func setup() {
	_devs = map[string]*device.Device{}
	_configByXAAL = map[string]map[string]*configGroup{}
	_configByKNXState = map[string]*configGroup{}
	_configByKNXWrite = map[string]*configGroup{}

	// gw
	gw, _ := schemas.Gateway(config.GWXaalAddr)

	var addresses []string
	for i := range config.Devices {
		confDev := &config.Devices[i]
		if confDev.XaalAddr == "" {
			confDev.XaalAddr = utils.GetRandomUUID()
			logger.Module("main").WithField("addr", confDev.XaalAddr).Info("New device")
		}

		dev, err := schemas.DeviceFromType(confDev.Type, confDev.XaalAddr)

		if err != nil {
			logger.Module("main").WithError(err).Warn()
		} else {
			addresses = append(addresses, dev.Address)
			_devs[dev.Address] = dev

			if err := linkMethods(dev, confDev.Type); err != nil {
				logger.Module("main").WithError(err).Error()
			}

			setupDev(dev, confDev.Name)

			xaal.AddDevice(dev)
		}

		for i := range confDev.Groups {
			confGroup := &confDev.Groups[i]
			confGroup.device = confDev

			if confGroup.GrpAddrWrite != "" {
				group, err := cemi.NewGroupAddrString(confGroup.GrpAddrWrite)
				if err != nil {
					logger.Module("main").WithError(err).Warn()
					break
				}
				confGroup.groupWrite = &group
				_configByKNXWrite[confGroup.GrpAddrWrite] = confGroup
			}

			if confGroup.GrpAddrState != "" {
				_configByKNXState[confGroup.GrpAddrState] = confGroup
			}

			if _, in := _configByXAAL[confDev.XaalAddr]; !in {
				_configByXAAL[confDev.XaalAddr] = map[string]*configGroup{}
			}
			_configByXAAL[confDev.XaalAddr][confGroup.Attribute] = confGroup

		}
	}
	gw.SetAttributeValue("embedded", addresses)
	setupDev(gw, fmt.Sprintf("IP Gateway [%s]", config.GatewayIP))
	xaal.AddDevice(gw)
}

func updateFromXAAL(msg *message.Message) {
	// TODO send Xaal notifications to KNX bus
}

func updateFromKNX() {
	// The inbound channel is closed with the connection.
	for msg := range client.Inbound() {
		addrKNX := msg.Destination.String()
		logger.Module("main").WithField("addrKNX", addrKNX).Debug("Received KNX message from")
		if confGroup, in := _configByKNXState[addrKNX]; in {
			addrXAAL := confGroup.device.XaalAddr
			attribute := confGroup.Attribute
			typeXAAL := confGroup.device.Type
			logger.Module("main").WithField("group", addrKNX).Debug("KNX State group found, process notification")
			if err := processKNXEvent(addrXAAL, typeXAAL, attribute, msg.Data); err != nil {
				logger.Module("main").WithError(err).Error()
			}
		} else {
			logger.Module("main").WithField("group", addrKNX).Debug("KNX State group not in config, ignoring")
		}
	}
}
