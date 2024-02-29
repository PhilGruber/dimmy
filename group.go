package main

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"
)

type Group struct {
	Device
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

	fmt.Println("Devices: " + config["devices"])
	devices := strings.Split(config["devices"], ",")
	fmt.Printf("Found %d devices\n", len(devices))
	g.devices = make([]DeviceInterface, len(devices))
	i := 0
	for k := range allDevices {
		fmt.Println("Existing device: " + k)
	}
	for _, key := range devices {
		fmt.Println("Finding " + key)
		_, ok := allDevices[key]
		if ok {
			fmt.Println("Type: " + reflect.TypeOf(allDevices[key]).Name())
			dev, ok := allDevices[key]
			if ok {
				fmt.Println("\tAdding : " + key)
				g.devices[i] = dev
				i = i + 1
			} else {
				fmt.Println("\tinvalid type")
			}
		}
	}

	g.Type = "group"
	return g
}

func (g Group) getCurrent() float64 {
	var current float64
	current = 0
	for _, d := range g.devices {
		current = math.Max(float64(d.getCurrent()), float64(current))
	}
	return current
}

func (g Group) getMax() int {
	max := 0
	for _, d := range g.devices {
		max = int(math.Max(float64(d.getMax()), float64(max)))
	}
	return max
}

func (g Group) getMin() int {
	min := 0
	for _, d := range g.devices {
		min = int(math.Min(float64(d.getMin()), float64(min)))
	}
	return min
}

func (g Group) setCurrent(current float64) {
	for _, d := range g.devices {
		d.setCurrent(current)
	}
}

func (g Group) UpdateValue() (float64, bool) {
	update := false
	current := 0.0
	for _, d := range g.devices {
		c, b := d.UpdateValue()
		update = b && update
		current = math.Max(c, current)
	}
	return current, update
}

func (g *Group) processRequest(request SwitchRequest) {
	for _, d := range g.devices {
		d.processRequest(request)
	}
}
