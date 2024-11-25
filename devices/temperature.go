package devices

import (
	core "github.com/PhilGruber/dimmy/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"strconv"
)

type Temperature struct {
	Device
	HasHumidity bool
}

func MakeTemperature(config core.DeviceConfig) Temperature {
	t := Temperature{}
	t.Emoji = "🌡️"
	t.setBaseConfig(config)

	t.Current = 0
	t.Type = "temperature"
	t.HasHumidity = false

	t.Triggers = []string{"temperature", "humidity"}

	return t
}

func NewTemperature(config core.DeviceConfig) *Temperature {
	s := MakeTemperature(config)
	return &s
}

func (t *Temperature) GetMin() int {
	return 0
}

func (t *Temperature) GetMax() int {
	return 99
}

func (t *Temperature) PublishValue(mqtt mqtt.Client) {
}

func (t *Temperature) ProcessRequest(request core.SwitchRequest) {
}

func (t *Temperature) UpdateValue() (float64, bool) {
	return 0, false
}

func (t *Temperature) GetMqttStateTopic() string {
	return t.MqttTopic
}

func (t *Temperature) GetMessageHandler(channel chan core.SwitchRequest, temperature DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := string(mqttMessage.Payload())
		log.Printf("Received new temperature for %s: %s", t.Name, payload)
		temperature, err := strconv.ParseFloat(payload[:], 64)
		if err != nil {
			log.Println("Received invalid temperature " + payload[:] + ": " + err.Error())
			return
		}
		t.SetCurrent(temperature)
	}
}

func (t *Temperature) GetHumidity() float64 {
	return -1
}

func (t *Temperature) GetTriggerValue(trigger string) interface{} {
	if trigger == "temperature" {
		return t.GetCurrent()
	}
	if trigger == "humidity" {
		return t.GetHumidity()
	}
	return nil
}
