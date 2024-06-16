package devices

import (
	"github.com/PhilGruber/dimmy/core"
	"log"
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
	GetName() string
	GetTriggers() []string
	GetReceivers() []string
	SetReceiverValue(string, any)
	GetTriggerValue(string) any

	PublishValue(mqtt.Client)
	PollValue(mqtt.Client)
}

type Device struct {
	Name        string
	MqttTopic   string     `json:"-"`
	MqttState   string     `json:"-"`
	Current     float64    `json:"value"`
	LastChanged *time.Time `json:"-"`
	Type        string
	Hidden      bool
	Triggers    []string
	Receivers   []string
}

func (d *Device) GetName() string {
	return d.Name
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

func (d *Device) GetTriggers() []string {
	return d.Triggers
}

func (d *Device) GetReceivers() []string {
	return d.Receivers
}

func (d *Device) SetReceiverValue(key string, value any) {
}

func (d *Device) GetTriggerValue(key string) any {
	return nil
}
