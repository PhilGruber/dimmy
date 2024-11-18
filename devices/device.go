package devices

import (
	"github.com/PhilGruber/dimmy/core"
	"github.com/google/uuid"
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
	GetName() string
	GetLabel() string
	GetMax() int
	GetMin() int
	GetHidden() bool
	GetCurrent() float64
	setCurrent(float64)
	GetEmoji() string
	ProcessRequest(core.SwitchRequest)
	GetMessageHandler(chan core.SwitchRequest, DeviceInterface) mqtt.MessageHandler
	GetStateMessageHandler(chan core.SwitchRequest, DeviceInterface) mqtt.MessageHandler
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
	Label       string
	Emoji       string
	Triggers    []string
	Receivers   []string
}

func (d *Device) setBaseConfig(config core.DeviceConfig) {
	d.MqttTopic = config.Topic
	d.Current = 0
	if config.Emoji != "" {
		d.Emoji = config.Emoji
	}

	if config.Name != "" {
		d.Name = config.Name
	} else {
		d.Name = uuid.NewString()
	}

	if config.Label != "" {
		log.Println("Setting label to " + config.Label)
		d.Label = config.Label
	} else {
		log.Println("No label found, setting label to Name: " + d.Name)
		d.Label = d.Name
	}

	d.Hidden = false
	if config.Options != nil {
		if config.Options.Hidden != nil {
			d.Hidden = *config.Options.Hidden
		}
	}
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

func (d *Device) GetName() string {
	return d.Name
}

func (d *Device) GetLabel() string {
	if d.Label != "" {
		return d.Label
	}
	return d.Name
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

func (d *Device) GetHidden() bool {
	return d.Hidden
}

func (d *Device) GetEmoji() string {
	return d.Emoji
}
