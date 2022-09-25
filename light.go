package main

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/project-eria/eria-core"

	"github.com/rs/zerolog/log"
	"github.com/vapourismo/knx-go/knx/dpt"
)

type light struct {
	*configDevice
	*eria.EriaThing
}

func (l *light) linkHandlers() error {
	for _, capability := range l.Capabilities {
		switch capability {
		case "LightBasic":
			l.SetActionHandler("toggle", l.lampToggle)
		case "LightDimmer":
			l.SetActionHandler("fade", l.lampFade)
		default:
			return fmt.Errorf("'%s' capability hasn't been implemented yet", capability)
		}
	}
	for key, conf := range l.States {
		conf := conf
		switch key {
		case "on":
			conf.handler = l.processKNXOn
		case "brightness":
			conf.handler = l.processKNXBrightness
		default:
			return fmt.Errorf("'%s'state has not beeing implemented for notifications", key)
		}
		_groupByKNXState[conf.GrpAddr] = conf
	}
	return nil
}

func (l *light) lampToggle(data interface{}) (interface{}, error) {
	var newValue = !l.GetPropertyValue("on").(bool)
	l.lampOnOffSend(newValue)
	return newValue, nil
}

func (l *light) lampFade(data interface{}) (interface{}, error) {
	brightness := float32(data.(float64))
	if confGroup, in := l.Actions["dimmer"]; in {
		payload := dpt.DPT_5001(brightness).Pack()
		if confGroup.groupWrite != nil {
			log.Trace().Str("device", l.Ref).Float32("brightness", brightness).Msg("[main:lampFade] Dimming Lamp")
			if err := sendKNX(confGroup.groupWrite, payload); err != nil {
				log.Error().Str("device", l.Ref).Err(err).Msg("[main:lampFade]")
				return nil, err
			}
		} else {
			log.Error().Str("device", l.Ref).Msg("[main:lampFade] Missing write groupe configuration for 'dimmer'")
			return nil, errors.New("missing write groupe configuration for 'dimmer'")
		}
	} else {
		log.Error().Str("device", l.Ref).Msg("[main:lampFade] Missing 'dimmer' configuration")
		return nil, errors.New("missing 'dimmer' configuration")
	}
	return brightness, nil
}

func (l *light) lampOnOffSend(value bool) {
	if confGroup, in := l.Actions["on"]; in {
		data := dpt.DPT_1001(value).Pack()
		if err := sendKNX(confGroup.groupWrite, data); err != nil {
			log.Error().Err(err).Msg("[main:lampOnOffSend]")
		}
	} else {
		log.Warn().Str("device", l.Ref).Msg("[main:lampOnOffSend] Missing KNX group for 'light'")
	}
}

func (l *light) processKNXOn(data []byte, _ bool) error {
	log.Trace().Msg("[main] Received light 'on' notification")

	var unpackedData dpt.DPT_1001
	err := unpackedData.Unpack(data)
	if err != nil {
		return errors.New("Unpacking 'on' data has failed: " + err.Error())
	}
	value := (strings.ToLower(unpackedData.String()) == "on")
	l.SetPropertyValue("on", value)
	return nil
}

func (l *light) processKNXBrightness(data []byte, _ bool) error {
	log.Trace().Msg("[main] Received light 'brightness' notification")

	var unpackedData dpt.DPT_5001
	err := unpackedData.Unpack(data)
	if err != nil {
		return errors.New("Unpacking 'brightness' data has failed: " + err.Error())
	}
	value := uint(math.Round(float64(unpackedData))) //Fix for https://github.com/vapourismo/knx-go/issues/23
	l.SetPropertyValue("brightness", value)

	return nil
}