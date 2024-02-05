package main

import (
    "time"
    "strconv"
    "math"
    "encoding/json"

    mqtt "github.com/eclipse/paho.mqtt.golang"
)

type ZLight struct {
    Light
}

func NewZLight(config map[string]string) *ZLight {
    d := makeZLight(config)
    return &d
}

func makeZLight(config map[string]string) ZLight {
    d := ZLight{};
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

    d.Hidden = false
    if val, ok := config["hidden"]; ok {
        d.Hidden = (val == "true")
    }

    tt := time.Now()
    d.LastChanged = &tt
    d.Type = "zlight"
    return d
}

func (l *ZLight) PublishValue(mqtt mqtt.Client) {
    tt := time.Now()
    newVal := int(math.Round(l.Current))
    var state string
    if newVal != l.LastSent {
        l.LastChanged = &tt
        l.LastSent = newVal

        brightness := int(math.Round(l.Current * 2.5))
        if brightness > 0 {
            state = "ON"
        } else {
            state = "OFF"
        }

        msg := Zigbee2MqttSetMessage{
            State: state,
            Brightness: brightness,
        }
        s, _ := json.Marshal(msg)
        mqtt.Publish(l.MqttTopic + "/set", 0, false, s)
    }
}
