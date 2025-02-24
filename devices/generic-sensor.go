package devices

import (
	"encoding/json"
	"github.com/PhilGruber/dimmy/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"sync"
	"time"
)

type SensorValue struct {
	Value       any               `json:"value"`
	LastChanged time.Time         `json:"LastChanged"`
	History     map[time.Time]any `json:"History"`
}

type Sensor struct {
	Device

	fields     []string
	Values     map[string]*SensorValue `json:"Values"`
	hasHistory bool
	valueMutex *sync.RWMutex
}

func MakeSensor(config core.DeviceConfig) Sensor {
	s := Sensor{}
	s.setBaseConfig(config)
	s.MqttState = config.Topic

	s.Type = "sensor"

	s.hasHistory = false
	if config.Options != nil {
		if config.Options.Fields != nil {
			s.fields = *config.Options.Fields
			s.Triggers = s.fields
		}

		if config.Options.History != nil {
			s.hasHistory = *config.Options.History
		}
	}
	s.valueMutex = new(sync.RWMutex)

	s.Triggers = s.fields

	s.Values = make(map[string]*SensorValue)
	for _, field := range s.fields {
		s.Values[field] = &SensorValue{Value: nil, LastChanged: time.Unix(0, 0), History: make(map[time.Time]any)}
	}

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
	s.valueMutex.Lock()
	s.Values[field].Value = value
	s.Values[field].LastChanged = time.Now()
	s.valueMutex.Unlock()
	if s.hasHistory {
		s.addHistory(field, value)
	}
}

func (s Sensor) GetValue(field string) any {
	s.valueMutex.RLock()
	defer s.valueMutex.RUnlock()
	return s.Values[field].Value
}

func (s Sensor) GetMessageHandler(_ chan core.SwitchRequest, _ DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := mqttMessage.Payload()
		var data map[string]any
		err := json.Unmarshal(payload, &data)
		if err != nil {
			log.Printf("[%32s] Error: %s\n", s.Name, err.Error())
			return
		}

		s.parseDefaultValues(data)

		for _, field := range s.fields {
			if value, ok := data[field]; ok {
				log.Printf("[%32s] Received new %s: %v\n", s.Name, field, value)
				s.SetValue(field, value)
			}
		}
	}
}

func (s Sensor) addHistory(field string, value any) {
	s.Values[field].History[time.Now()] = value
}

func (s Sensor) UpdateValue() (float64, bool) { return 0, false }

func (s Sensor) GetTriggerValue(trigger string) interface{} {
	if s.HasField(trigger) {
		return s.GetValue(trigger)
	}
	return nil
}

func (s Sensor) ClearTrigger(trigger string) {
	if s.HasField(trigger) {
		s.SetValue(trigger, nil)
	}
}
