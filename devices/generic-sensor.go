package devices

import (
	"encoding/json"
	"github.com/PhilGruber/dimmy/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
)

type Sensor struct {
	Device

	fields []string
	Values map[string]any `json:"Values"`
}

func MakeSensor(config core.DeviceConfig) Sensor {
	s := Sensor{}
	s.setBaseConfig(config)
	s.MqttState = config.Topic

	s.Type = "sensor"

	if config.Options != nil && config.Options.Fields != nil {
		s.fields = *config.Options.Fields
	}

	s.Values = make(map[string]any)

	return s
}

func NewSensor(config core.DeviceConfig) *Sensor {
	s := MakeSensor(config)
	return &s
}

func (s Sensor) GetFields() []string {
	return s.fields
}

func (s Sensor) HasField(field string) bool {
	for _, f := range s.fields {
		if f == field {
			return true
		}
	}
	return false
}

func (s Sensor) SetValue(field string, value any) {
	s.Values[field] = value
}

func (s Sensor) GetValue(field string) any {
	return s.Values[field]
}

func (s Sensor) GetMessageHandler(_ chan core.SwitchRequest, _ DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := mqttMessage.Payload()
		var data map[string]any
		err := json.Unmarshal(payload, &data)
		if err != nil {
			log.Println("Error: " + err.Error())
			return
		}

		s.parseDefaultValues(data)

		for _, field := range s.fields {
			if value, ok := data[field]; ok {
				log.Printf("[%s] Received new value for %s: %v\n", s.Name, field, value)
				s.SetValue(field, value)
			}
		}
	}
}

func (s Sensor) UpdateValue() (float64, bool) { return 0, false }
