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

func NewGroup(config core.DeviceConfig, allDevices map[string]DeviceInterface) *Group {
	g := Group{}

	g.Hidden = false
	deviceType := ""
	var devices []string
	if config.Options != nil {
		if config.Options.Hidden != nil {
			g.Hidden = *config.Options.Hidden
		}

		devices = strings.Split(*config.Options.Devices, ",")
		g.devices = make([]DeviceInterface, len(devices))
	}

	i := 0
	for _, key := range devices {
		_, ok := allDevices[key]
		if ok {
			dev, ok := allDevices[key]
			if ok {
				if deviceType == "" {
					deviceType = dev.GetType()
				} else if deviceType != dev.GetType() {
					fmt.Println("Can't add device " + key + " to group, as it is of a different type than the other devices in the group")
					return nil
				}
				g.devices[i] = dev
				i = i + 1
			}
		} else {
			fmt.Println("Could not find device " + key + ", as part of a group")
		}
	}

	tt := time.Now()
	g.LastChanged = &tt

	g.Type = "group"
	return &g
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
