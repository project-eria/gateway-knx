package main

import (
	"time"

	"github.com/vapourismo/knx-go/knx/dpt"
)

/* For Watermeter we use 12.001 as default KNX DTP */

func watermeterNotification(address string, attribute string, data []byte) {
	var (
		unpackedData dpt.DPT_12001
		attributes   = make(map[string]interface{})
	)
	err := unpackedData.Unpack(data)
	if err != nil {
		return
	}
	attributes[attribute] = unpackedData
	attributes["timestamp"] = time.Now().Unix()
	sendXAAL(address, attributes)
}
