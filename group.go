package main

import (
	"fmt"
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

func (g *Group) getCurrent() float64 {
	var current float64
	current = 0
	for _, d := range g.devices {
		current = math.Max(d.getCurrent(), current)
	}
	return current
}

func (g *Group) getMax() int {
	groupMax := 0
	for _, d := range g.devices {
		groupMax = int(math.Max(float64(d.getMax()), float64(groupMax)))
	}
	return groupMax
}

func (g *Group) getMin() int {
	groupMin := 0
	for _, d := range g.devices {
		groupMin = int(math.Min(float64(d.getMin()), float64(groupMin)))
	}
	return groupMin
}

func (g *Group) processRequest(request SwitchRequest) {
	for _, d := range g.devices {
		d.processRequest(request)
	}
	g.processRequestChild(request)
}

func (g *Group) UpdateValue() (float64, bool) {
	changed := false
	for _, d := range g.devices {
		_, thisChanged := d.UpdateValue()
		if thisChanged {
			changed = true
		}
	}
	g.setCurrent(g.getCurrent())
	return g.getCurrent(), changed
}
