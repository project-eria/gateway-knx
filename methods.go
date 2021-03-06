package main

import (
	"fmt"

	logger "github.com/project-eria/eria-logger"
	"github.com/project-eria/xaal-go"
	"github.com/project-eria/xaal-go/device"

	"github.com/vapourismo/knx-go/knx/cemi"

	"github.com/vapourismo/knx-go/knx"
)

func linkMethods(dev *device.Device, typeXAAL string) error {
	switch typeXAAL {
	case "lamp.basic":
		dev.HandleMethod("on", lampOn)
		dev.HandleMethod("off", lampOff)
	case "lamp.dimmer":
		dev.HandleMethod("on", lampOn)
		dev.HandleMethod("off", lampOff)
		dev.HandleMethod("dim", lampDim)
	case "shutter.basic":
		dev.HandleMethod("up", shutterUp)
		dev.HandleMethod("down", shutterDown)
		dev.HandleMethod("stop", shutterStop)
	case "shutter.position":
		dev.HandleMethod("up", shutterUp)
		dev.HandleMethod("down", shutterDown)
		dev.HandleMethod("stop", shutterStop)
		dev.HandleMethod("position", shutterPosition)
	case "watermeter.basic":
	default:
		return fmt.Errorf("%s type methods hasn't been implemented yet", typeXAAL)
	}
	return nil
}

func processKNXEvent(addrXAAL string, typeXAAL string, attribute string, data []byte) error {
	var err error
	switch typeXAAL {
	case "lamp.basic", "lamp.dimmer":
		err = lampNotification(addrXAAL, attribute, data)
	case "shutter.basic", "shutter.position":
		err = shutterNotification(addrXAAL, attribute, data)
	case "watermeter.basic":
		err = watermeterNotification(addrXAAL, attribute, data)
	default:
		return fmt.Errorf("%s type notifications hasn't been implemented yet", typeXAAL)
	}
	return err
}

func sendXAAL(address string, attributes map[string]interface{}) {
	device := _devs[address]
	for attribute, value := range attributes {
		device.SetAttributeValue(attribute, value)
	}
	xaal.NotifyAttributesChange(device)
}

func sendKNX(group *cemi.GroupAddr, data []byte) error {
	logger.Module("main").WithField("group", group).Debug("Sending KNX")

	event := knx.GroupEvent{
		Command:     knx.GroupWrite,
		Destination: *group,
		Data:        data,
	}

	return client.Send(event)
}
