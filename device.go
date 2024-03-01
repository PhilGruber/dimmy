package main

import (
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type DeviceInterface interface {
	UpdateValue() (float64, bool)
	getTimeoutRequest() (SwitchRequest, bool)
	generateRequest(string) (SwitchRequest, bool)

	getMqttTopic() string
	getType() string
	getMax() int
	getMin() int
	getCurrent() float64
	setCurrent(float64)
	processRequest(SwitchRequest)
	getMessageHandler(chan SwitchRequest, DeviceInterface) mqtt.MessageHandler

	PublishValue(mqtt.Client)
}

type Device struct {
	MqttTopic   string     `json:"-"`
	Current     float64    `json:"value"`
	LastChanged *time.Time `json:"-"`
	Type        string
	Hidden      bool
}

func (d *Device) getCurrent() float64 {
	return d.Current
}

func (d *Device) setCurrent(current float64) {
	d.Current = current
}

func (d *Device) getType() string {
	return d.Type
}

func (d *Device) getMqttTopic() string {
	return d.MqttTopic
}

func (d *Device) getTimeoutRequest() (SwitchRequest, bool) {
	var r SwitchRequest
	return r, false
}

func (d *Device) generateRequest(cmd string) (SwitchRequest, bool) {
	var r SwitchRequest
	return r, false
}

func (d *Device) PublishValue(mqtt mqtt.Client) {
}

func (d *Device) getMessageHandler(channel chan SwitchRequest, sensor DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
	}
}
