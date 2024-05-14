package devices

import (
	core "github.com/PhilGruber/dimmy/core"
	"log"
	"strconv"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Thermostat struct {
	Device

	TargetDevice      string
	TargetTemperature float64
	Margin            float64
}

func MakeThermostat(config core.DeviceConfig) Thermostat {
	t := Thermostat{}
	t.MqttTopic = config.Topic
	if config.Options == nil {
		log.Fatalf("Thermostat %s does not have any options", config.Name)
	}
	if config.Options.Target == nil {
		log.Fatalf("Thermostat %s does not have a target device", config.Name)
	}
	t.TargetDevice = *config.Options.Target

	t.Margin = 0.5
	if config.Options.Margin != nil {
		t.Margin = *config.Options.Margin
	}

	t.TargetTemperature = 18
	t.Current = 0
	t.Type = "thermostat"
	return t
}

func (t *Thermostat) GetMin() int {
	return 0
}

func (t *Thermostat) GetMax() int {
	return 99
}

func NewThermostat(config core.DeviceConfig) *Thermostat {
	s := MakeThermostat(config)
	return &s
}

func (t *Thermostat) PublishValue(mqtt mqtt.Client) {
}

func (t *Thermostat) ProcessRequest(request core.SwitchRequest) {
	t.TargetTemperature = request.Value
}

func (t *Thermostat) GenerateRequest(payload string) (core.SwitchRequest, bool) {

	t.Current, _ = strconv.ParseFloat(payload, 64)

	var request core.SwitchRequest
	if t.Current < (t.TargetTemperature - t.Margin/2) {
		log.Printf("Too cold, turning on %s. Currently %.2f C, need %.2f C", t.TargetDevice, t.Current, t.TargetTemperature-t.Margin/2)
		request.Device = t.TargetDevice
		request.Value = 1
		request.Duration = 0

		return request, true

	} else if t.Current > (t.TargetTemperature + t.Margin/2) {
		log.Printf("Too hot, turning off %s. Currently %.2f C, need %.2f C", t.TargetDevice, t.Current, t.TargetTemperature+t.Margin/2)
		request.Device = t.TargetDevice
		request.Value = 0
		request.Duration = 0

		return request, true
	}

	return request, false
}

func (t *Thermostat) UpdateValue() (float64, bool) {
	return 0, false
}

func (t *Thermostat) GetMqttStateTopic() string {
	return t.MqttTopic
}

func (t *Thermostat) getMessageHandler(channel chan core.SwitchRequest, thermostat DeviceInterface) mqtt.MessageHandler {
	log.Println("Subscribing to " + thermostat.GetMqttTopic())
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := string(mqttMessage.Payload())
		log.Println("Received new temperature: " + string(payload))

		if request, ok := thermostat.GenerateRequest(payload[:]); ok {
			channel <- request
		}
	}
}
