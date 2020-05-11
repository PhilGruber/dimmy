package main

import (
    "time"

    mqtt "github.com/eclipse/paho.mqtt.golang"
)

type DeviceInterface interface {
    UpdateValue() (float64, bool)
    getTimeoutRequest() (SwitchRequest, bool)
    generateMotionRequest(string) SwitchRequest

    getMqttTopic() string
    getType() string
    getMax() int
    getMin() int
    getCurrent() float64
    setCurrent(float64)
    getStep() float64
    setStep(float64)
    getTarget() int
    setTarget(int)
    getLastSent() int
    setLastSent(int)
    getLastChanged() *time.Time
    setLastChanged(*time.Time)
    processRequest(SwitchRequest)
}

type Device struct {
    MqttTopic string `json:"-"`
    Current float64 `json:"value"`
    LastChanged *time.Time `json:"-"`
    Type string
    Hidden bool
}

func (d Device) getCurrent() float64 {
    return d.Current
}

func (d *Device) setCurrent(current float64) {
    d.Current = current
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

func (d Device) generateMotionRequest(cmd string) SwitchRequest {
    var r SwitchRequest
    return r
}

func (d Device) PublishValue(mqtt mqtt.Client) {
}

