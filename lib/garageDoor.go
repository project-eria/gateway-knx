package lib

import (
	"errors"
	"fmt"
	"strings"

	"github.com/project-eria/eria-core"
	"github.com/project-eria/go-wot/producer"
	zlog "github.com/rs/zerolog/log"
	"github.com/vapourismo/knx-go/knx/dpt"
)

type garageDoor struct {
	*ConfigDevice
	producer.ExposedThing
}

func (d *garageDoor) linkSetup() error {
	producer := eria.Producer("")
	producer.PropertyUseDefaultHandlers(d, "open")
	producer.PropertyUseDefaultHandlers(d, "moving")
	producer.PropertyUseDefaultHandlers(d, "closed")
	d.SetActionHandler("open", d.doorOpen)
	d.SetActionHandler("close", d.doorClose)

	for key, conf := range d.States {
		conf := conf
		switch key {
		case "moving":
			conf.handler = d.processKNXMoving
		case "open":
			conf.handler = d.processKNXOpen
		case "closed":
			conf.handler = d.processKNXClosed
		default:
			return fmt.Errorf("'%s'state has not beeing implemented for notifications", key)
		}
		_groupByKNXState[conf.GrpAddr] = conf
		d.requestKNXState(key) // Requesting initial state value
	}
	return nil
}

func (d *garageDoor) doorOpen(data interface{}, parameters map[string]interface{}) (interface{}, error) {
	d.doorTrigger("open")
	return nil, nil
}

func (d *garageDoor) doorClose(data interface{}, parameters map[string]interface{}) (interface{}, error) {
	d.doorTrigger("close")
	return nil, nil
}

func (d *garageDoor) doorTrigger(action string) {
	if confGroup, in := d.Actions[action]; in {
		data := dpt.DPT_1001(true).Pack()
		if err := writeKNX(confGroup.GroupWrite, data); err != nil {
			zlog.Error().Str("device", d.ID).Str("action", action).Err(err).Msg("[main:doorTrigger]")
		}
	} else {
		zlog.Warn().Str("device", d.ID).Str("action", action).Msg("[main:doorTrigger] Missing KNX group")
	}
}

func (d *garageDoor) processKNXMoving(data []byte, invertValue bool) error {
	zlog.Trace().Msg("[main] Received door 'moving' notification")

	var unpackedData dpt.DPT_1001
	err := unpackedData.Unpack(data)
	if err != nil {
		return errors.New("Unpacking 'moving' data has failed: " + err.Error())
	}
	value := (strings.ToLower(unpackedData.String()) == "on")
	if invertValue {
		value = !value
	}
	p := eria.Producer("")
	p.SetPropertyValue(d, "moving", value)
	if value {
		// Door is moving: leaves both end-of-travel contacts
		p.SetPropertyValue(d, "open", false)
		p.SetPropertyValue(d, "closed", false)
	}
	return nil
}

func (d *garageDoor) processKNXOpen(data []byte, invertValue bool) error {
	zlog.Trace().Msg("[main] Received door 'open' notification")

	var unpackedData dpt.DPT_1001
	err := unpackedData.Unpack(data)
	if err != nil {
		return errors.New("Unpacking 'open' data has failed: " + err.Error())
	}
	value := (strings.ToLower(unpackedData.String()) == "on")
	if invertValue {
		value = !value
	}
	eria.Producer("").SetPropertyValue(d, "open", value)
	return nil
}

func (d *garageDoor) processKNXClosed(data []byte, invertValue bool) error {
	zlog.Trace().Msg("[main] Received door 'closed' notification")

	var unpackedData dpt.DPT_1001
	err := unpackedData.Unpack(data)
	if err != nil {
		return errors.New("Unpacking 'closed' data has failed: " + err.Error())
	}
	value := (strings.ToLower(unpackedData.String()) == "on")
	if invertValue {
		value = !value
	}
	eria.Producer("").SetPropertyValue(d, "closed", value)
	return nil
}
