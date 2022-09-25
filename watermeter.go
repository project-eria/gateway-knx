package main

import (
	"errors"
	"fmt"

	"github.com/project-eria/eria-core"

	"github.com/rs/zerolog/log"
	"github.com/vapourismo/knx-go/knx/dpt"
)

/* For Watermeter we use 12.001 as default KNX DTP */

type watermeter struct {
	*configDevice
	*eria.EriaThing
}

func (w *watermeter) linkHandlers() error {
	for key, conf := range w.States {
		conf := conf
		switch key {
		case "liters":
			conf.handler = w.processKNXLiters
		default:
			return fmt.Errorf("'%s'state has not beeing implemented for notifications", key)
		}
		_groupByKNXState[conf.GrpAddr] = conf

	}
	return nil
}

func (w *watermeter) processKNXLiters(data []byte, _ bool) error {
	log.Trace().Msg("[main] Received watermeter 'liters' notification")

	var unpackedData dpt.DPT_12001
	err := unpackedData.Unpack(data)
	if err != nil {
		return errors.New("Unpacking 'liters' data has failed: " + err.Error())
	}
	value := float64(unpackedData)
	w.SetPropertyValue("liters", value)
	return nil
}
