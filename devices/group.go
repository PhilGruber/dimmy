package devices

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/PhilGruber/dimmy/core"
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
		return nil
	}

	if config.Options.Devices == nil {
		log.Println("Group " + config.Name + " has no devices")
		return nil
	}

	g.devices = make([]DeviceInterface, 0, len(*config.Options.Devices))
	for _, key := range *config.Options.Devices {
		dev, ok := allDevices[key]
		if !ok || dev == nil {
			log.Printf("Could not find device %s, as part of group %s", key, config.Name)
			continue
		}
		if len(g.devices) == 0 {
			g.Type = dev.GetType()
			g.Icon = dev.GetEmoji()
			g.MqttState = dev.GetMqttStateTopic()
			g.Receivers = dev.GetReceivers()
		} else if g.Type != dev.GetType() {
			log.Printf("Can't add device %s to group %s: type %s differs from %s", key, config.Name, dev.GetType(), g.Type)
			return nil
		}
		g.devices = append(g.devices, dev)
	}
	if len(g.devices) == 0 {
		log.Printf("Group %s has no available devices", config.Name)
		return nil
	}

	g.init()

	log.Printf("[%32s] Created group with %d devices\n", config.Name, len(g.devices))

	return &g
}

func (g *Group) GetCurrent() float64 {
	var current float64 = 0
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
	if len(request.Value) > 0 && (request.Value[0] == '+' || request.Value[0] == '-') {
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
