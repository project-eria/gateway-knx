package lib

import (
	"errors"
	"fmt"
	"math"

	"github.com/project-eria/eria-core"
	"github.com/project-eria/go-wot/producer"
	zlog "github.com/rs/zerolog/log"
	"github.com/vapourismo/knx-go/knx/dpt"
)

type shutter struct {
	*ConfigDevice
	producer.ExposedThing
}

func (s *shutter) linkSetup() error {
	producer := eria.Producer("")
	switch s.Type {
	case "ShutterBasic":
		producer.PropertyUseDefaultHandlers(s, "open")
		s.SetActionHandler("open", s.shutterOpen)
		s.SetActionHandler("close", s.shutterClose)
		//s.SetActionHandler("stop", s.shutterStop)
	case "ShutterPosition":
		producer.PropertyUseDefaultHandlers(s, "open")
		producer.PropertyUseDefaultHandlers(s, "position")
		s.SetActionHandler("open", s.shutterOpen)
		s.SetActionHandler("close", s.shutterClose)
		//s.SetActionHandler("stop", s.shutterStop)
		s.SetActionHandler("setPosition", s.shutterSetPosition)
	default:
		return fmt.Errorf("'%s' type hasn't been implemented yet", s.Type)
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
		s.requestKNXState(key) // Requesting initial state value
	}
	return nil
}

func (s *shutter) shutterOpen(data interface{}, parameters map[string]interface{}) (interface{}, error) {
	s.shutterSend(false)
	return nil, nil
}

func (s *shutter) shutterClose(data interface{}, parameters map[string]interface{}) (interface{}, error) {
	s.shutterSend(true)
	return nil, nil
}

// func (s *shutter) shutterStop(data interface{}) (interface{}, error) {
// 	//TODO shutterSend(request., false)
// }

func (s *shutter) shutterSend(value bool) {
	if confGroup, in := s.Actions["open"]; in {
		data := dpt.DPT_1009(value).Pack()
		if err := writeKNX(confGroup.GroupWrite, data); err != nil {
			zlog.Error().Str("device", s.ID).Err(err).Msg("[main:shutterSend]")
		}
	} else {
		zlog.Warn().Str("device", s.ID).Msg("[main:shutterSend] Missing KNX group for 'open'")
	}
}

func (s *shutter) shutterSetPosition(data interface{}, parameters map[string]interface{}) (interface{}, error) {
	target := float32(data.(int))
	targetEffective := target
	if confGroup, in := s.Actions["set"]; in {
		if confGroup.InvertValue {
			targetEffective = 100 - targetEffective // Invert 0%=>Close 100%=>Open
		}
		data := dpt.DPT_5001(targetEffective).Pack()
		if confGroup.GroupWrite != nil {
			zlog.Trace().Str("device", s.ID).Float32("targetEffective", targetEffective).Msg("[main:shutterPosition] Moving Shutter")

			if err := writeKNX(confGroup.GroupWrite, data); err != nil {
				zlog.Error().Err(err).Msg("[main:shutterPosition]")
			}
		} else {
			zlog.Error().Msg("[main:shutterPosition] Missing write groupe configuration for 'set'")
		}
	} else {
		zlog.Error().Msg("[main:shutterPosition] Missing 'set' configuration")
	}
	return nil, nil
}

// func (s *shutter) processKNXOpen(data []byte, invertValue bool) error {
// 	zlog.Trace().Msg("[main] Received shutter 'opening' notification")

// 	var unpackedData dpt.DPT_1009
// 	err := unpackedData.Unpack(data)
// 	if err != nil {
// 		return errors.New("Unpacking 'open' data has failed: " + err.Error())
// 	}
// 	value := (strings.ToLower(unpackedData.String()) == "open")

// 	zlog.Trace().Bool("value", value).Msg("[main] openning value")
// 	s.SetPropertyValue("open", value)
// 	return nil
// }

func (s *shutter) processKNXPosition(data []byte, invertValue bool) error {
	zlog.Trace().Msg("[main] Received shutter 'position' notification")

	var unpackedData dpt.DPT_5001
	err := unpackedData.Unpack(data)
	if err != nil {
		return errors.New("Unpacking 'position' data has failed: " + err.Error())
	}
	value := int(math.Round(float64(unpackedData))) //Fix for https://github.com/vapourismo/knx-go/issues/23
	if invertValue {
		value = 100 - value // Invert 0%=>Close 100%=>Open
	}
	zlog.Trace().Int("value", value).Msg("[main] position value")
	eria.Producer("").SetPropertyValue(s, "position", value)
	eria.Producer("").SetPropertyValue(s, "open", (value > 0))
	return nil
}
