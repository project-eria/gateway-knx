package lib

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/project-eria/eria-core"
	"github.com/project-eria/go-wot/producer"
	zlog "github.com/rs/zerolog/log"
	"github.com/vapourismo/knx-go/knx/dpt"
)

const triggerActiveDuration = 1 * time.Second

type trigger struct {
	*ConfigDevice
	producer.ExposedThing

	mu    sync.Mutex
	timer *time.Timer
}

func (t *trigger) linkSetup() error {
	eria.Producer("").PropertyUseDefaultHandlers(t, "triggered")

	for key, conf := range t.States {
		conf := conf
		switch key {
		case "triggered":
			conf.handler = t.processKNXTriggered
		default:
			return fmt.Errorf("'%s'state has not beeing implemented for notifications", key)
		}
		_groupByKNXState[conf.GrpAddr] = conf
	}
	return nil
}

func (t *trigger) processKNXTriggered(data []byte, invertValue bool) error {
	zlog.Trace().Msg("[main] Received trigger 'triggered' notification")

	var unpackedData dpt.DPT_1001
	err := unpackedData.Unpack(data)
	if err != nil {
		return errors.New("Unpacking 'triggered' data has failed: " + err.Error())
	}
	value := (strings.ToLower(unpackedData.String()) == "on")
	if invertValue {
		value = !value
	}
	if !value {
		// Only care about the rising edge — falling edges are handled by the auto-reset timer.
		return nil
	}

	p := eria.Producer("")
	p.SetPropertyValue(t, "triggered", true)

	t.mu.Lock()
	if t.timer != nil {
		t.timer.Stop()
	}
	t.timer = time.AfterFunc(triggerActiveDuration, func() {
		eria.Producer("").SetPropertyValue(t, "triggered", false)
	})
	t.mu.Unlock()

	return nil
}
