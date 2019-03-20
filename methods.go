package main

import (
	"fmt"

	"github.com/project-eria/xaal-go/device"
	"github.com/project-eria/xaal-go/engine"

	"github.com/vapourismo/knx-go/knx/cemi"

	"github.com/vapourismo/knx-go/knx"
)

func linkMethods(dev *device.Device, typeXAAL string) error {
	switch typeXAAL {
	case "lamp.basic":
		dev.AddMethod("on", lampOn)
		dev.AddMethod("off", lampOff)
		break
	case "shutter.basic":
		dev.AddMethod("up", shutterUp)
		dev.AddMethod("down", shutterDown)
		dev.AddMethod("stop", shutterStop)
		break
	default:
		return fmt.Errorf("%s type methods hasn't been implemented yet", typeXAAL)
	}
	return nil
}

func processKNXEvent(addrXAAL string, typeXAAL string, attribute string, data []byte) error {

	switch typeXAAL {
	case "lamp.basic":
		lampNotification(addrXAAL, attribute, data)
		break
	case "shutter.basic":
		shutterNotification(addrXAAL, attribute, data)
		break
	default:
		return fmt.Errorf("%s type notifications hasn't been implemented yet", typeXAAL)
	}
	return nil
}

func sendXAAL(address string, attribute string, value string) {
	device := _devs[address]
	device.SetAttributeValue(attribute, value)
	engine.NotifyAttributesChange(device)
}

func sendKNX(group cemi.GroupAddr, data []byte) error {
	event := knx.GroupEvent{
		Command:     knx.GroupWrite,
		Destination: group,
		Data:        data,
	}

	return client.Send(event)
}
