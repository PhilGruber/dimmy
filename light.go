package main

import (
    "time"
    "strconv"
    "math"

    mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Light struct {
    Device
    Dimmable
    LastSent int `json:"-"`
}

func makeLight(config map[string]string) Light {
    d := Light{};
    d.MqttTopic = config["topic"]
    d.Current = 0
    d.Target = 0
    min, ok := config["min"]
    if !ok {
        min = "0"
    }
    max, ok := config["max"]
    if !ok {
        max = "100"
    }
    d.Min, _ = strconv.Atoi(min)
    d.Max, _ = strconv.Atoi(max)
    tt := time.Now()
    d.LastChanged = &tt
    d.Type = "light"
    return d
}

func NewLight(config map[string]string) *Light {
    d := makeLight(config)
    return &d
}

func (l *Light) PublishValue(mqtt mqtt.Client) {
    tt := time.Now()
    newVal := int(math.Round(l.Current))
    if newVal != l.LastSent {
        l.LastChanged = &tt
        l.LastSent = newVal
        mqtt.Publish(l.MqttTopic, 0, false, strconv.Itoa(int(math.Round(l.Current))))
    }
}
