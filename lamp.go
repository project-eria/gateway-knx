package main

import (
	"strings"

	"github.com/project-eria/xaal-go/device"

	logger "github.com/project-eria/eria-logger"
	"github.com/vapourismo/knx-go/knx/dpt"
)

/* For lamp we use 1.001 as default KNX DTP */

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

func lampNotification(address string, attribute string, data []byte) {
	var (
		unpackedData dpt.DPT_1001
		attributes   = make(map[string]interface{})
	)
	err := unpackedData.Unpack(data)
	if err != nil {
		return
	}
	attributes[attribute] = strings.ToLower(unpackedData.String())
	sendXAAL(address, attributes)
}
