package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"

	"github.com/project-eria/xaal-go/device"
	"github.com/project-eria/xaal-go/engine"
	"github.com/project-eria/xaal-go/message"
	"github.com/project-eria/xaal-go/schemas"
	"github.com/project-eria/xaal-go/utils"

	"github.com/project-eria/config-manager"
	"github.com/project-eria/logger"
	"github.com/vapourismo/knx-go/knx/cemi"

	"github.com/vapourismo/knx-go/knx"
)

func version() string {
	return fmt.Sprintf("0.0.1 - %s (engine commit %s)", engine.Timestamp, engine.GitCommit)
}

const configFile = "gateway-knx.json"

func setupDev(dev *device.Device) {
	dev.VendorID = "ERIA"
	dev.ProductID = "KNXIPGateway"
	dev.Info = "gateway.knx"
	dev.Version = version()
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
	GrpAddr   string
	DPT       string
	Attribute string
	group     cemi.GroupAddr
	device    *configDevice
}

var _devs map[string]*device.Device

// For direct access
var _configByXAAL map[string]map[string]*configGroup
var _configByKNX map[string]*configGroup

var client knx.GroupTunnel

func main() {
	defer os.Exit(0)
	_showVersion := flag.Bool("v", false, "Display the version")
	if !flag.Parsed() {
		flag.Parse()
	}

	// Show version (-v)
	if *_showVersion {
		fmt.Println(version())
		os.Exit(0)
	}

	logger.Module("main").Infof("Starting Gateway KNX %s...", version())

	// Loading config
	cm, err := configmanager.Init(configFile, &config)
	if err != nil {
		if configmanager.IsFileMissing(err) {
			err = cm.Save()
			if err != nil {
				logger.Module("main").WithField("filename", configFile).Fatal(err)
			}
			logger.Module("main").Fatal("JSON Config file do not exists, created...")
		} else {
			logger.Module("main").WithField("filename", configFile).Fatal(err)
		}
	}

	if err := cm.Load(); err != nil {
		logger.Module("main").Fatal(err)
	}
	defer cm.Close()

	engine.Init()

	setup()

	// Save for new Address during setup
	cm.Save()

	engine.AddRxHandler(updateFromXAAL)

	// Connect to the KNX gateway.
	GWAddr := fmt.Sprintf("%s:%d", config.GatewayIP, config.GatewayPort)
	client, err = knx.NewGroupTunnel(GWAddr, knx.DefaultTunnelConfig)
	if err != nil {
		logger.Module("main").WithError(err).Fatal("Can connect KNX IP Gateway")
	}

	// Close upon exiting. Even if the gateway closes the connection, we still have to clean up.
	defer client.Close()

	// Launch the xAAL engine
	go engine.Run()
	defer engine.Stop()

	// Receive messages from the gateway
	go updateFromKNX()

	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// Block until keyboard interrupt is received.
	<-c
	runtime.Goexit()
}

// setup : create devices, register ...
func setup() {
	_devs = map[string]*device.Device{}
	_configByXAAL = map[string]map[string]*configGroup{}
	_configByKNX = map[string]*configGroup{}

	// gw
	gw := schemas.Gateway(config.GWXaalAddr)

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

			setupDev(dev)
			engine.AddDevice(dev)
		}
		for i := range confDev.Groups {
			confGroup := &confDev.Groups[i]
			confGroup.device = confDev
			group, err := cemi.NewGroupAddrString(confGroup.GrpAddr)
			if err != nil {
				logger.Module("main").WithError(err).Warn()
			} else {
				confGroup.group = group
				_configByKNX[confGroup.GrpAddr] = confGroup
				if _, in := _configByXAAL[confDev.XaalAddr]; !in {
					_configByXAAL[confDev.XaalAddr] = map[string]*configGroup{}
				}
				_configByXAAL[confDev.XaalAddr][confGroup.Attribute] = confGroup
			}
		}
	}
	gw.SetAttributeValue("embedded", addresses)
	setupDev(gw)
	engine.AddDevice(gw)
}

func updateFromXAAL(msg *message.Message) {
	// TODO send Xaal notifications to KNX bus
}

func updateFromKNX() {
	// The inbound channel is closed with the connection.
	for msg := range client.Inbound() {
		//fmt.Printf("%+v\n", msg)
		addrKNX := msg.Destination.String()

		if confGroup, in := _configByKNX[addrKNX]; in {
			addrXAAL := confGroup.device.XaalAddr
			attribute := confGroup.Attribute
			typeKNX := confGroup.DPT
			typeXAAL := confGroup.device.Type
			if err := processKNXEvent(addrXAAL, typeXAAL, attribute, typeKNX, msg.Data); err != nil {
				logger.Module("main").WithError(err).Error()
			}
		} else {
			logger.Module("main").WithField("group", addrKNX).Debug("KNX group not in config")
		}
	}
}
