package main

import (
	"errors"
	"fmt"
	"math"

	"github.com/project-eria/eria-core"

	"github.com/rs/zerolog/log"
	"github.com/vapourismo/knx-go/knx/dpt"
)

type shutter struct {
	*configDevice
	*eria.EriaThing
}

func (s *shutter) linkHandlers() error {
	for _, capability := range s.Capabilities {
		switch capability {
		case "ShutterBasic":
			s.SetActionHandler("open", s.shutterOpen)
			s.SetActionHandler("close", s.shutterClose)
			//s.SetActionHandler("stop", s.shutterStop)
		case "ShutterPosition":
			s.SetActionHandler("setPosition", s.shutterSetPosition)
		default:
			return fmt.Errorf("'%s' capability hasn't been implemented yet", capability)
		}
	}

	for key, conf := range s.States {
		conf := conf
		switch key {
		// case "opening":
		// 	conf.handler = s.processKNXOpen
		case "position":
			conf.handler = s.processKNXPosition
		default:
			return fmt.Errorf("'%s'state has not beeing implemented for notifications", key)
		}
		_groupByKNXState[conf.GrpAddr] = conf
	}
	return nil
}

func (s *shutter) shutterOpen(data interface{}) (interface{}, error) {
	s.shutterSend(false)
	return nil, nil
}

func (s *shutter) shutterClose(data interface{}) (interface{}, error) {
	s.shutterSend(true)
	return nil, nil
}

// func (s *shutter) shutterStop(data interface{}) (interface{}, error) {
// 	//TODO shutterSend(request., false)
// }

func (s *shutter) shutterSend(value bool) {
	if confGroup, in := s.Actions["open"]; in {
		data := dpt.DPT_1009(value).Pack()
		if err := sendKNX(confGroup.groupWrite, data); err != nil {
			log.Error().Str("device", s.Ref).Err(err).Msg("[main:shutterSend]")
		}
	} else {
		log.Warn().Str("device", s.Ref).Msg("[main:shutterSend] Missing KNX group for 'open'")
	}
}

func (s *shutter) shutterSetPosition(data interface{}) (interface{}, error) {
	target := float32(data.(float64))
	targetEffective := target
	if confGroup, in := s.Actions["position"]; in {
		if confGroup.InvertValue {
			targetEffective = 100 - targetEffective // Invert 0%=>Close 100%=>Open
		}
		data := dpt.DPT_5001(targetEffective).Pack()
		if confGroup.groupWrite != nil {
			log.Trace().Str("device", s.Ref).Float32("targetEffective", targetEffective).Msg("[main:shutterPosition] Moving Shutter")

			if err := sendKNX(confGroup.groupWrite, data); err != nil {
				log.Error().Err(err).Msg("[main:shutterPosition]")
			}
		} else {
			log.Error().Msg("[main:shutterPosition] Missing write groupe configuration for 'position'")
		}
	} else {
		log.Error().Msg("[main:shutterPosition] Missing 'position' configuration")
	}
	return nil, nil
}

// func (s *shutter) processKNXOpen(data []byte, invertValue bool) error {
// 	log.Trace().Msg("[main] Received shutter 'opening' notification")

// 	var unpackedData dpt.DPT_1009
// 	err := unpackedData.Unpack(data)
// 	if err != nil {
// 		return errors.New("Unpacking 'open' data has failed: " + err.Error())
// 	}
// 	value := (strings.ToLower(unpackedData.String()) == "open")

// 	log.Trace().Bool("value", value).Msg("[main] openning value")
// 	s.SetPropertyValue("open", value)
// 	return nil
// }

func (s *shutter) processKNXPosition(data []byte, invertValue bool) error {
	log.Trace().Msg("[main] Received shutter 'position' notification")

	var unpackedData dpt.DPT_5001
	err := unpackedData.Unpack(data)
	if err != nil {
		return errors.New("Unpacking 'position' data has failed: " + err.Error())
	}
	value := int(math.Round(float64(unpackedData))) //Fix for https://github.com/vapourismo/knx-go/issues/23
	if invertValue {
		value = 100 - value // Invert 0%=>Close 100%=>Open
	}
	log.Trace().Int("value", value).Msg("[main] position value")
	s.SetPropertyValue("position", value)
	s.SetPropertyValue("open", (value > 0))
	return nil
}
