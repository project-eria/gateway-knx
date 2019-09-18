package main

import (
	"fmt"
	"math"
	"strings"

	"github.com/project-eria/xaal-go/device"

	logger "github.com/project-eria/eria-logger"
	"github.com/vapourismo/knx-go/knx/dpt"
)

/* For lamp we use 1.001 as default KNX DTP */

func lampOn(dev *device.Device, args map[string]interface{}) map[string]interface{} {
	lampOnOffSend(dev.Address, true)
	return nil
}

func lampOff(dev *device.Device, args map[string]interface{}) map[string]interface{} {
	lampOnOffSend(dev.Address, false)
	return nil
}

func lampDim(dev *device.Device, args map[string]interface{}) map[string]interface{} {
	value, ok := args["target"]
	if ok {
		valueInt := float32(value.(float64))
		if confGroup, in := _configByXAAL[dev.Address]["dimmer"]; in {
			data := dpt.DPT_5001(valueInt).Pack()
			if confGroup.groupWrite != nil {
				logger.Module("main:lamp").WithFields(logger.Fields{"address": dev.Address, "target": valueInt}).Debug("Dimming Lamp")
				if err := sendKNX(confGroup.groupWrite, data); err != nil {
					logger.Module("main:lamp").Error(err)
				}
			} else {
				logger.Module("main:lamp").Error("Missing write groupe configuration for 'dimmer'")
			}
		} else {
			logger.Module("main:lamp").Error("Missing 'dimmer' configuration")
		}
	} else {
		logger.Module("main:lamp").Error("Missing 'target' parameter")
	}
	return nil
}

func lampOnOffSend(address string, value bool) {
	if confGroup, in := _configByXAAL[address]["light"]; in {
		data := dpt.DPT_1001(value).Pack()
		if err := sendKNX(confGroup.groupWrite, data); err != nil {
			logger.Module("main:lamp").Error(err)
		}
	} else {
		logger.Module("main:lamp").WithField("addr", address).Warn("Missing KNX group for 'light'")
	}
}

func lampNotification(address string, attribute string, data []byte) error {
	var attributes = make(map[string]interface{})
	switch attribute {
	case "light":
		var unpackedData dpt.DPT_1001
		err := unpackedData.Unpack(data)
		if err != nil {
			return fmt.Errorf("Unpacking '%s' data has failed (%s)", attribute, err)
		}
		attributes["light"] = strings.ToLower(unpackedData.String())
	case "dimmer":
		var unpackedData dpt.DPT_5001
		err := unpackedData.Unpack(data)
		if err != nil {
			return fmt.Errorf("Unpacking '%s' data has failed (%s)", attribute, err)
		}
		dataInt := int(math.Round(float64(unpackedData))) //Fix for https://github.com/vapourismo/knx-go/issues/23
		attributes["dimmer"] = dataInt
	default:
		return fmt.Errorf("Notification for '%s' attribute is not implemented", attribute)
	}
	sendXAAL(address, attributes)
	return nil
}
