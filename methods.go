package main

import (
	"errors"

	"github.com/project-eria/eria-core"
	zlog "github.com/rs/zerolog/log"

	"github.com/vapourismo/knx-go/knx"
	"github.com/vapourismo/knx-go/knx/cemi"
)

type knxThing interface {
	linkHandlers() error
}

func newKNXThing(config *configDevice, t *eria.EriaThing) (knxThing, error) {
	zlog.Info().Str("device", config.Ref).Msg("[main] new KNX Thing")

	var knxthing knxThing
	switch config.Type {
	case "LightBasic", "LightDimmer":
		knxthing = &light{
			configDevice: config,
			EriaThing:    t}
	case "ShutterBasic", "ShutterPosition":
		knxthing = &shutter{
			configDevice: config,
			EriaThing:    t}
	case "WaterMeter":
		knxthing = &watermeter{
			configDevice: config,
			EriaThing:    t}
	default:
		return nil, errors.New(config.Type + " type hasn't been implemented yet")
	}

	if err := knxthing.linkHandlers(); err != nil {
		zlog.Error().Err(err).Msg("[main]")
	}

	return knxthing, nil
}

func sendKNX(group *cemi.GroupAddr, data []byte) error {
	zlog.Trace().Stringer("group", group).Msg("[main] Sending KNX")

	event := knx.GroupEvent{
		Command:     knx.GroupWrite,
		Destination: *group,
		Data:        data,
	}

	return client.Send(event)
}
