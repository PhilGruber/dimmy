package devices

import (
	"fmt"
	core "github.com/PhilGruber/dimmy/core"
	"math"
	"strings"
	"time"
)

type Group struct {
	Dimmable
	devices []DeviceInterface
}

func NewGroup(config map[string]string, allDevices map[string]DeviceInterface) *Group {
	d := makeGroup(config, allDevices)
	return &d
}

func makeGroup(config map[string]string, allDevices map[string]DeviceInterface) Group {
	g := Group{}

	g.Hidden = false
	if val, ok := config["hidden"]; ok {
		g.Hidden = val == "true"
	}

	tt := time.Now()
	g.LastChanged = &tt

	devices := strings.Split(config["devices"], ",")
	g.devices = make([]DeviceInterface, len(devices))
	i := 0
	for _, key := range devices {
		_, ok := allDevices[key]
		if ok {
			dev, ok := allDevices[key]
			if ok {
				g.devices[i] = dev
				i = i + 1
			}
		} else {
			fmt.Println("Could not find device " + key + ", as part of a group")
		}
	}

	g.Type = "group"
	return g
}

func (g *Group) GetCurrent() float64 {
	var current float64
	current = 0
	for _, d := range g.devices {
		current = math.Max(d.GetCurrent(), current)
	}
	return current
}

func (g *Group) GetMax() int {
	groupMax := 0
	for _, d := range g.devices {
		groupMax = int(math.Max(float64(d.GetMax()), float64(groupMax)))
	}
	return groupMax
}

func (g *Group) GetMin() int {
	groupMin := 0
	for _, d := range g.devices {
		groupMin = int(math.Min(float64(d.GetMin()), float64(groupMin)))
	}
	return groupMin
}

func (g *Group) ProcessRequest(request core.SwitchRequest) {
	for _, d := range g.devices {
		d.ProcessRequest(request)
	}
	g.ProcessRequestChild(request)
}

func (g *Group) UpdateValue() (float64, bool) {
	changed := false
	for _, d := range g.devices {
		_, thisChanged := d.UpdateValue()
		if thisChanged {
			changed = true
		}
	}
	g.setCurrent(g.GetCurrent())
	return g.GetCurrent(), changed
}
