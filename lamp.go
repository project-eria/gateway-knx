package main

import (
	"strings"

	"github.com/project-eria/xaal-go/device"

	"github.com/project-eria/logger"
	"github.com/vapourismo/knx-go/knx/dpt"
)

func lampOn(dev *device.Device, args map[string]interface{}) map[string]interface{} {
	lampSend(dev.Address, true)
	return nil
}

func lampOff(dev *device.Device, args map[string]interface{}) map[string]interface{} {
	lampSend(dev.Address, false)
	return nil
}

func lampSend(address string, value bool) {
	if confGroup, in := _configByXAAL[address]["light"]; in {
		data := dpt.DPT_1001(value).Pack()

		if err := sendKNX(confGroup.group, data); err != nil {
			logger.Module("main").Error(err)
		}
	}
}

func lampNotification(address string, attribute string, dptType string, data []byte) {
	if dptType == "DPT_1001" {
		var unpackedData dpt.DPT_1001
		err := unpackedData.Unpack(data)
		if err != nil {
			return
		}
		value := strings.ToLower(unpackedData.String())
		sendXAAL(address, attribute, value)
	}
}
