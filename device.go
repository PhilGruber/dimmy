package main

import (
    "time"
    "strconv"
)

type Device struct {
    MqttTopic string `json:"-"`
    Current float64 `json:"value"`
    Target int `json:"target"`
    Step float64 `json:"-"`
    Min int `json:"-"`
    Max int `json:"-"`
    LastChanged *time.Time `json:"-"`
}

func makeDevice(config map[string]string) Device {
    d := Device{};
    d.MqttTopic = config["topic"]
    d.Current = 0
    d.Target = 0
    min, ok := config["min"]
    if !ok {
        min = "0"
    }
    max, ok := config["max"]
    if !ok {
        max = "0"
    }
    d.Min, _ = strconv.Atoi(min)
    d.Max, _ = strconv.Atoi(max)
    tt := time.Now()
    d.LastChanged = &tt
    return d
}

func NewDevice(config map[string]string) *Device {
    d := makeDevice(config)
    return &d
}

func (d Device) UpdateValue() (float64, bool) {
    if d.Current != float64(d.Target) {
        if (d.Current > float64(d.Target)) {
            d.Current -= d.Step
            if (d.Current < float64(d.Target)) {
                d.Current = float64(d.Target)
            }
        } else {
            d.Current += d.Step
            if (d.Current > float64(d.Target)) {
                d.Current = float64(d.Target)
            }
        }
        return d.Current, true

    }
    return 0, false
}

