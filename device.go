package main

import (
    "time"
    "strconv"
    "math"

    mqtt "github.com/eclipse/paho.mqtt.golang"
)

type DeviceInterface interface {
    UpdateValue() bool
    PublishValue(mqtt.Client)
    getTimeoutRequest() (SwitchRequest, bool)
    getMotionRequest(string) SwitchRequest

    getMqttTopic() string
    getType() string
    getMax() int
    getMin() int
    getCurrent() float64
    getStep() float64
    setStep(float64)
    getTarget() int
    setTarget(int)
}

type Device struct {
    MqttTopic string `json:"-"`
    Current float64 `json:"value"`
    Target int `json:"target"`
    Step float64 `json:"-"`
    Min int `json:"-"`
    Max int `json:"-"`
    LastChanged *time.Time `json:"-"`
    Type string
}

func (d Device) getMin() int {
    return d.Min
}

func (d Device) getMax() int {
    return d.Max
}

func (d Device) getStep() float64 {
    return d.Step
}

func (d *Device) setStep(step float64) {
    d.Step = step
}

func (d Device) getTarget() int {
    return d.Target
}

func (d *Device) setTarget(target int) {
    d.Target = target
}

func (d Device) getCurrent() float64 {
    return d.Current
}

func (d Device) getType() string {
    return d.Type
}

func (d Device) getMqttTopic() string {
    return d.MqttTopic
}

func (d Device) getTimeoutRequest() (SwitchRequest, bool) {
    var r SwitchRequest
    return r, false
}

func (d Device) getMotionRequest(cmd string) SwitchRequest {
    var r SwitchRequest
    return r
}

func (d *Device) UpdateValue() bool {
    if d.Current != float64(d.Target) {
        if (d.Current > float64(d.Target)) {
            d.Current -= d.Step
            if (d.Current <= float64(d.Target)) {
                d.Current = float64(d.Target)
            }
        } else {
            d.Current += d.Step
            if (d.Current >= float64(d.Target)) {
                d.Current = float64(d.Target)
            }
        }
        return true
    }

    return false
}

func (d Device) PublishValue(mqtt mqtt.Client) {
}


type Light struct {
    Device
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

func (l Light) PublishValue(mqtt mqtt.Client) {
    tt := time.Now()
    if int(math.Round(l.Current)) != l.LastSent {
        l.LastChanged = &tt
//      log.Printf("Setting %s to %f", int(math.Round(value)))
        l.LastSent = int(math.Round(l.Current))
        mqtt.Publish(l.MqttTopic, 0, false, strconv.Itoa(int(math.Round(l.Current))))
    }
}
