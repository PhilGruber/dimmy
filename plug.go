package main

import (
    "time"
    "strconv"
    "math"

    mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Plug struct {
    Device
    needsSending bool
    Min int `json:"-"`
    Max int `json:"-"`
}

func makePlug(config map[string]string) Plug {
    p := Plug{};
    p.MqttTopic = config["topic"]
    p.Current = 0

    p.Hidden = false
    if val, ok := config["hidden"]; ok {
        p.Hidden = (val == "true")
    }

    tt := time.Now()
    p.LastChanged = &tt
    p.Type = "plug"
    p.needsSending = false
    p.Min = 0
    p.Max = 1
    return p
}

func NewPlug(config map[string]string) *Plug {
    p := makePlug(config)
    return &p
}

func (p *Plug) PublishValue(mqtt mqtt.Client) {
    mqtt.Publish(p.MqttTopic, 0, false, strconv.Itoa(int(math.Round(p.Current))))
}

func (p *Plug) UpdateValue() (float64, bool) {
    if p.needsSending {
        return p.Current, true
    }
    return 0, false
}

func (p *Plug) getMax() int {
    return p.Max
}

func (p *Plug) getMin() int {
    return p.Min
}

func (p *Plug) processRequest(request SwitchRequest) {
    if float64(request.Value) != p.Current {
        p.Current = float64(request.Value)
        if p.Current > 1 {
            p.Current = 1
        }
        if p.Current < 0 {
            p.Current = 0
        }
        p.needsSending = true
    }
}
