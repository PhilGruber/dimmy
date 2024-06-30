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
	g.setBaseConfig(config)

	g.Type = ""

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

	g.devices = make([]DeviceInterface, len(*config.Options.Devices))

	i := 0
	for _, key := range *config.Options.Devices {
		_, ok := allDevices[key]
		if ok {
			dev, ok := allDevices[key]
			if ok {
				if g.Type == "" {
					g.Type = dev.GetType()
				} else if g.Type != dev.GetType() {
					log.Println("Can't add device " + key + " to group, as it is of a different type than the other devices in the group")
					fmt.Printf("%s != %s", g.Type, dev.GetType())
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
	return 100
}

func (g *Group) GetMin() int {
	return 0
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
