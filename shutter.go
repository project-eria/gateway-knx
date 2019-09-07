package main

import (
	"fmt"
	"math"
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
		if confGroup, in := _configByXAAL[dev.Address]["position"]; in {
			data := dpt.DPT_5001(valueInt).Pack()
			logger.Module("main:lamp").WithFields(logger.Fields{"address": dev.Address, "target": valueInt}).Debug("Moving Shutter")
			if err := sendKNX(confGroup.group, data); err != nil {
				logger.Module("main:shutter").Error(err)
			}
		}
	} else {
		logger.Module("main:shutter").Error("Missing 'position' parameter")
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

func shutterNotification(address string, attribute string, data []byte) error {
	var attributes = make(map[string]interface{})
	switch attribute {
	case "action":
		var unpackedData dpt.DPT_1009
		err := unpackedData.Unpack(data)
		if err != nil {
			return fmt.Errorf("Unpacking '%s' data has failed (%s)", attribute, err)
		}
		attributes["action"] = strings.ToLower(unpackedData.String())
	case "position":
		var unpackedData dpt.DPT_5001
		err := unpackedData.Unpack(data)
		if err != nil {
			return fmt.Errorf("Unpacking '%s' data has failed (%s)", attribute, err)
		}
		fmt.Println(unpackedData)
		fmt.Printf("%#v %+v\n", unpackedData, unpackedData)
		fmt.Println(int(unpackedData))
		dataInt := int(math.Round(float64(unpackedData))) //Fix for https://github.com/vapourismo/knx-go/issues/23
		attributes["position"] = dataInt
	default:
		return fmt.Errorf("Notification for '%s' attribute is not implemented", attribute)
	}
	sendXAAL(address, attributes)
	return nil
}
