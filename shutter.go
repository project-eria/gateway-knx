package main

import (
	"strings"

	logger "github.com/project-eria/eria-logger"
	"github.com/project-eria/xaal-go/device"

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

func shutterPosition(dev *device.Device, args map[string]interface{}) map[string]interface{} {
	value, ok := args["target"]
	if ok {
		valueInt := float32(value.(float64))
		if confGroup, in := _configByXAAL[dev.Address]["positionTarget"]; in {
			data := dpt.DPT_5001(valueInt).Pack()
			logger.Module("main:lamp").WithFields(logger.Fields{"address": dev.Address, "target": valueInt}).Debug("Moving Shutter")
			if err := sendKNX(confGroup.group, data); err != nil {
				logger.Module("main:shutter").Error(err)
			}
		}
	} else {
		logger.Module("main:shutter").Error("Missing 'positionTarget' parameter")
	}
	return nil
}

func shutterSend(address string, value bool) {
	if confGroup, in := _configByXAAL[address]["action"]; in {
		data := dpt.DPT_1009(value).Pack()

		if err := sendKNX(confGroup.group, data); err != nil {
			logger.Module("main:shutter").Error(err)
		}
	}
}

func shutterNotification(address string, attribute string, data []byte) {
	var (
		unpackedData dpt.DPT_1009
		attributes   = make(map[string]interface{})
	)
	err := unpackedData.Unpack(data)
	if err != nil {
		return
	}
	attributes[attribute] = strings.ToLower(unpackedData.String())
	sendXAAL(address, attributes)
}
