package main

import (
    "time"
)

type Device struct {
    MqttTopic string `json:"-"`
    Current int `json:"value"`
    Target int `json:"target"`
    Delay time.Duration `json:"-"`
    LastChanged *time.Time `json:"-"`
}

func makeDevice(config map[string]string) Device {
    d := Device{};
    d.MqttTopic = config["topic"]
    d.Current = 0
    d.Target = 0
    d.Delay = 0
    tt := time.Now()
    d.LastChanged = &tt
    return d
}

func NewDevice(config map[string]string) *Device {
    d := Device{};
    d.MqttTopic = config["topic"]
    d.Current = 0
    d.Target = 0
    d.Delay = 0
    tt := time.Now()
    d.LastChanged = &tt
    return &d
}

func (d Device) UpdateValue() (int, bool) {
    if d.Current != d.Target {
        now := time.Now()
        if d.LastChanged.Add(d.Delay).Before(now) {
            if (d.Current > d.Target) {
                d.Current--
            } else {
                d.Current++
            }
            return d.Current, true
        }
    }
    return 0, false
}

