package devices

import (
	"fmt"
	"github.com/PhilGruber/dimmy/core"
	"log"
	"math"
	"strconv"
	"time"
)

type Group struct {
	Dimmable
	devices []DeviceInterface
}

func NewGroup(config core.DeviceConfig, allDevices map[string]DeviceInterface) *Group {
	g := Group{}
	g.Icon = "🏠"
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
					g.Icon = dev.GetEmoji()
					g.MqttState = dev.GetMqttStateTopic()
					g.Receivers = dev.GetReceivers()
				} else if g.Type != dev.GetType() {
					log.Println("Can't add Device " + key + " to group, as it is of a different type than the other devices in the group")
					fmt.Printf("%s != %s", g.Type, dev.GetType())
					return nil
				}
				g.devices[i] = dev
				i = i + 1
			}
		} else {
			fmt.Println("Could not find Device " + key + ", as part of a group")
		}
	}

	g.init()

	log.Printf("[%32s] Created group with %d devices\n", config.Name, len(g.devices))

	return &g
}

func (g *Group) GetCurrent() float64 {
	var current float64
	current = 0
	for _, d := range g.devices {
		current = math.Max(d.GetCurrent(), current)
	}
	g.SetCurrent(current)
	return current
}

func (g *Group) GetMax() int {
	return 100
}

func (g *Group) GetMin() int {
	return 0
}

func (g *Group) ProcessRequest(request core.SwitchRequest) {
	if request.Value[0] == '+' || request.Value[0] == '-' {
		value, err := strconv.ParseFloat(request.Value, 64)
		if err == nil {
			request.Value = fmt.Sprintf("%f", g.GetCurrent()+value)
		}
	}
	for _, d := range g.devices {
		d.ProcessRequest(request)
	}
	g.ProcessRequestChild(request)
}
