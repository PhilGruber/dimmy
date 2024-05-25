package devices

import (
	"fmt"
	core "github.com/PhilGruber/dimmy/core"
	"log"
	"math"
	"time"
)

type Group struct {
	Dimmable
	devices []DeviceInterface
}

func NewGroup(config core.DeviceConfig, allDevices map[string]DeviceInterface) *Group {
	g := Group{}

	g.Type = "group"
	g.Hidden = false
	deviceType := ""

	tt := time.Now()
	g.LastChanged = &tt

	if config.Options == nil {
		log.Println("Group " + config.Name + " has no options")
		return &g
	}

	if config.Options == nil || config.Options.Devices == nil {
		log.Println("Group " + config.Name + " has no devices")
		return &g
	}

	if config.Options.Hidden != nil {
		g.Hidden = *config.Options.Hidden
	}

	g.devices = make([]DeviceInterface, len(*config.Options.Devices))

	i := 0
	for _, key := range *config.Options.Devices {
		_, ok := allDevices[key]
		if ok {
			dev, ok := allDevices[key]
			if ok {
				if deviceType == "" {
					deviceType = dev.GetType()
				} else if deviceType != dev.GetType() {
					log.Println("Can't add device " + key + " to group, as it is of a different type than the other devices in the group")
					fmt.Printf("%s != %s", deviceType, dev.GetType())
					return nil
				}
				g.devices[i] = dev
				i = i + 1
			}
		} else {
			fmt.Println("Could not find device " + key + ", as part of a group")
		}
	}

	log.Println("Created group " + config.Name + " with " + fmt.Sprint(len(g.devices)) + " devices")

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
