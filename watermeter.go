package main

import (
	"fmt"
	"strconv"

	"github.com/vapourismo/knx-go/knx/dpt"
)

/* For Watermeter we use 12.001 as default KNX DTP */

func watermeterNotification(address string, attribute string, data []byte) error {
	var attributes = make(map[string]interface{})
	switch attribute {
	case "liters":
		var unpackedData dpt.DPT_12001
		err := unpackedData.Unpack(data)
		if err != nil {
			return fmt.Errorf("Unpacking '%s' data has failed (%s)", attribute, err)
		}
		attributes["liters"] = strconv.Itoa(int(unpackedData))
	default:
		return fmt.Errorf("Notification for '%s' attribute is not implemented", attribute)
	}
	sendXAAL(address, attributes)
	return nil
}
