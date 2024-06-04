package devices

import (
	core "github.com/PhilGruber/dimmy/core"
	"log"
	"math"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type DeviceInterface interface {
	UpdateValue() (float64, bool)
	GetTimeoutRequest() (core.SwitchRequest, bool)
	GenerateRequest(string) (core.SwitchRequest, bool)

	GetMqttTopic() string
	GetMqttStateTopic() string
	GetType() string
	GetMax() int
	GetMin() int
	GetCurrent() float64
	setCurrent(float64)
	ProcessRequest(core.SwitchRequest)
	GetMessageHandler(chan core.SwitchRequest, DeviceInterface) mqtt.MessageHandler
	GetStateMessageHandler(chan core.SwitchRequest, DeviceInterface) mqtt.MessageHandler

	PublishValue(mqtt.Client)
	PollValue(mqtt.Client)
}

type Device struct {
	DeviceInterface
	MqttTopic   string     `json:"-"`
	MqttState   string     `json:"-"`
	Current     float64    `json:"value"`
	LastChanged *time.Time `json:"-"`
	Type        string
	Hidden      bool
}

func (d *Device) GetCurrent() float64 {
	return d.Current
}

func (d *Device) setCurrent(current float64) {
	d.Current = current
}

func (d *Device) GetType() string {
	return d.Type
}

func (d *Device) GetMqttTopic() string {
	return d.MqttTopic
}

func (d *Device) GetMqttStateTopic() string {
	return d.MqttState
}

func (d *Device) GetTimeoutRequest() (core.SwitchRequest, bool) {
	var r core.SwitchRequest
	return r, false
}

func (d *Device) GenerateRequest(cmd string) (core.SwitchRequest, bool) {
	var r core.SwitchRequest
	return r, false
}

func (d *Device) PublishValue(mqtt.Client) {
}

func (d *Device) PollValue(mqtt.Client) {
}

func (d *Device) GetMessageHandler(channel chan core.SwitchRequest, sensor DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
	}
}

func (d *Device) GetStateMessageHandler(channel chan core.SwitchRequest, sensor DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		log.Println("Received state message from " + d.MqttState)
	}
}

func (d *Device) PercentageToValue(percentage float64) int {
	if percentage <= 1.0 {
		return d.GetMin() + int(math.Round(percentage))
	}
	return d.GetMin() + 1 + int(float64(d.GetMax()-d.GetMin()-1)*(percentage-1)/99)
}

func (d *Device) ValueToPercentage(value int) float64 {
	if value <= d.GetMin() {
		return 0
	}
	if value <= d.GetMin()+1 {
		return 1
	}
	if value >= d.GetMax() {
		return 100
	}
	return 1 + float64(value-d.GetMin()-1)*99/float64(d.GetMax()-d.GetMin()-1)
}
