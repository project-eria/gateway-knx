package lib

import (
	"errors"

	"github.com/project-eria/eria-core"
	zlog "github.com/rs/zerolog/log"

	"github.com/vapourismo/knx-go/knx"
	"github.com/vapourismo/knx-go/knx/cemi"
)

type ConfigDevice struct {
	Type    string                         `yaml:"type"`
	Name    string                         `yaml:"name"`
	Ref     string                         `yaml:"ref"`
	States  map[string]*configStatesGroup  `yaml:"states"`
	Actions map[string]*configActionsGroup `yaml:"actions"`
}

type configStatesGroup struct {
	InvertValue bool   `yaml:"invertValue" default:"false"`
	GrpAddr     string `yaml:"grpAddr"`
	handler     func([]byte, bool) error
	GroupRead   *cemi.GroupAddr
}

type configActionsGroup struct {
	InvertValue bool   `yaml:"invertValue" default:"false"`
	GrpAddr     string `yaml:"grpAddr"`
	GroupWrite  *cemi.GroupAddr
}

type knxThing interface {
	linkSetup() error
}

// For direct access
var _groupByKNXState map[string]*configStatesGroup = map[string]*configStatesGroup{}
var client knx.GroupTunnel

func ConnectKNX(gatewayAddr string, config knx.TunnelConfig) {
	var err error
	client, err = knx.NewGroupTunnel(gatewayAddr, config)
	if err != nil {
		zlog.Fatal().Err(err).Msg("[main] Can connect KNX IP Gateway")
	}
}

func CloseKNX() {
	client.Close()
}

func NewKNXThing(config *ConfigDevice, t *eria.EriaThing) (knxThing, error) {
	zlog.Info().Str("device", config.Ref).Msg("[main] new KNX Thing")

	var knxthing knxThing
	switch config.Type {
	case "LightBasic", "LightDimmer":
		knxthing = &light{
			ConfigDevice: config,
			EriaThing:    t}
	case "ShutterBasic", "ShutterPosition":
		knxthing = &shutter{
			ConfigDevice: config,
			EriaThing:    t}
	case "WaterMeter":
		knxthing = &watermeter{
			ConfigDevice: config,
			EriaThing:    t}
	default:
		return nil, errors.New(config.Type + " type hasn't been implemented yet")
	}

	if err := knxthing.linkSetup(); err != nil {
		zlog.Error().Err(err).Msg("[main]")
	}

	return knxthing, nil
}

func writeKNX(group *cemi.GroupAddr, data []byte) error {
	zlog.Trace().Stringer("group", group).Msg("[main] Sending GroupWrite KNX")

	event := knx.GroupEvent{
		Command:     knx.GroupWrite,
		Destination: *group,
		Data:        data,
	}

	return client.Send(event)
}

func readKNX(group *cemi.GroupAddr) error {
	zlog.Trace().Stringer("group", group).Msg("[main] Sending GroupRead KNX")

	event := knx.GroupEvent{
		Command:     knx.GroupRead,
		Destination: *group,
	}

	return client.Send(event)
}

func (c *ConfigDevice) requestKNXState(state string) {
	if confGroup, in := c.States[state]; in {
		if err := readKNX(confGroup.GroupRead); err != nil {
			zlog.Error().Str("device", c.Ref).Err(err).Msg("[main:requestKNXState]")
		}
	} else {
		zlog.Warn().Str("device", c.Ref).Str("state", state).Msg("[main:requestKNXState] Missing KNX group")
	}
}

func UpdateFromKNX() {
	// The inbound channel is closed with the connection.
	for msg := range client.Inbound() {
		addrKNX := msg.Destination.String()
		zlog.Trace().Str("addrKNX", addrKNX).Msg("[main] Received KNX message from")
		if confGroup, in := _groupByKNXState[addrKNX]; in {
			zlog.Trace().Str("group", addrKNX).Msg("[main] KNX State group found, process notification")
			if err := confGroup.handler(msg.Data, confGroup.InvertValue); err != nil {
				zlog.Error().Err(err).Msg("[main]")
			}
		} else {
			zlog.Trace().Str("group", addrKNX).Msg("[main] KNX State group not in config, ignoring")
		}
	}
}
