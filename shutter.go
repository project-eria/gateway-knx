package main

import (
	"strings"

	"github.com/project-eria/xaal-go/device"

	"github.com/project-eria/logger"
	"github.com/vapourismo/knx-go/knx/dpt"
)

/* For shutter we use 1.009 as default KNX DTP */
func shutterUp(dev *device.Device, args map[string]interface{}) map[string]interface{} {
	shutterSend(dev.Address, false)
	return nil
}

func shutterDown(dev *device.Device, args map[string]interface{}) map[string]interface{} {
	shutterSend(dev.Address, true)
	return nil
}

func shutterStop(dev *device.Device, args map[string]interface{}) map[string]interface{} {
	//TODO shutterSend(dev.Address, false)
	return nil
}

func shutterSend(address string, value bool) {
	if confGroup, in := _configByXAAL[address]["action"]; in {
		data := dpt.DPT_1009(value).Pack()

		if err := sendKNX(confGroup.group, data); err != nil {
			logger.Module("main").Error(err)
		}
	}
}

func shutterNotification(address string, attribute string, data []byte) {
	var unpackedData dpt.DPT_1009
	err := unpackedData.Unpack(data)
	if err != nil {
		return
	}
	value := strings.ToLower(unpackedData.String())
	sendXAAL(address, attribute, value)
}
