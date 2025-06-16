package devices

import (
	"encoding/json"
	"fmt"
	"github.com/PhilGruber/dimmy/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"sync"
	"time"
)

type SensorValue struct {
	Value       any             `json:"value"`
	LastChanged time.Time       `json:"LastChanged"`
	Since       *time.Time      `json:"Since"`
	History     []SensorHistory `json:"History"`
}

type SensorHistory struct {
	Time  time.Time `json:"Time"`
	Value any       `json:"Value"`
}

type Sensor struct {
	Device

	sensors    []core.Sensor
	Values     map[string]*SensorValue `json:"Values"`
	hasHistory bool
	valueMutex *sync.RWMutex
}

func NewSensor(config core.DeviceConfig) *Sensor {
	s := Sensor{}
	s.setBaseConfig(config)
	s.MqttState = config.Topic

	s.Type = "sensor"

	s.hasHistory = false
	if config.Options != nil {
		if config.Options.Sensors != nil && len(*config.Options.Sensors) > 0 {
			s.sensors = make([]core.Sensor, len(*config.Options.Sensors))
			for i, sensorConfig := range *config.Options.Sensors {
				s.sensors[i] = sensorConfig
				s.Triggers = append(s.Triggers, sensorConfig.Name)
			}
		} else {
			if config.Options.Fields != nil {
				for _, field := range *config.Options.Fields {
					s.sensors = append(s.sensors, core.Sensor{Name: field, Hidden: false})
					s.Triggers = append(s.Triggers, field)
				}
			}
		}

		if config.Options.History != nil {
			s.hasHistory = *config.Options.History
		}
	}

	for i := range s.sensors {
		if s.sensors[i].Icon != "" {
			continue // Icon is already set, skip default assignment
		}
		if s.sensors[i].Name == "humidity" {
			s.sensors[i].Icon = "💧"
		}
		if s.sensors[i].Name == "temperature" {
			s.sensors[i].Icon = "🌡️"
		}
		if s.sensors[i].Name == "illuminance" {
			s.sensors[i].Icon = "🔆"
		}
		if s.sensors[i].Name == "button" {
			s.sensors[i].Icon = "🔘"
		}
		if s.sensors[i].Name == "action" {
			s.sensors[i].Icon = "⚙️"
		}
		if s.sensors[i].Name == "presence" {
			s.sensors[i].Icon = "🧍"
		}
		if s.sensors[i].Name == "vibration" {
			s.sensors[i].Icon = "📳"
		}
	}

	if s.Icon == "" && len(s.sensors) > 0 {
		s.Icon = s.sensors[0].Icon
	}

	s.valueMutex = new(sync.RWMutex)

	s.Values = make(map[string]*SensorValue)
	for _, sensor := range s.sensors {
		s.Values[sensor.Name] = &SensorValue{Value: nil, LastChanged: time.Unix(0, 0), History: make([]SensorHistory, 0)}
	}

	return &s
}

func (s *Sensor) GetFields() []string {
	fields := make([]string, 0, len(s.sensors))
	for _, sensor := range s.sensors {
		fields = append(fields, sensor.Name)
	}
	return fields
}

func (s *Sensor) GetSensors() []core.Sensor {
	return s.sensors
}

func (s *Sensor) HasField(field string) bool {
	for _, s := range s.sensors {
		if s.Name == field {
			return true
		}
	}
	return false
}

func (s *Sensor) SetValue(field string, value any) {
	s.valueMutex.Lock()
	s.Values[field].Value = value
	s.Values[field].LastChanged = time.Now()

	for _, sensor := range s.sensors {
		if sensor.ShowSince != nil && sensor.Name == field {
			if fmt.Sprintf("%v", value) == fmt.Sprintf("%v", *sensor.ShowSince) {
				now := time.Now()
				s.Values[field].Since = &now
			}
		}
	}

	s.valueMutex.Unlock()
	if s.hasHistory {
		s.addHistory(field, value)
	}

	s.UpdateRules(field, value)
}

func (s *Sensor) GetValue(field string) any {
	s.valueMutex.RLock()
	defer s.valueMutex.RUnlock()
	return s.Values[field].Value
}

func (s *Sensor) GetMessageHandler(_ chan core.SwitchRequest, _ DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := mqttMessage.Payload()
		var data map[string]any
		err := json.Unmarshal(payload, &data)
		if err != nil {
			log.Printf("[%32s] Error: %s\n", s.Name, err.Error())
			return
		}

		s.parseDefaultValues(data)

		for _, sensor := range s.sensors {
			if value, ok := data[sensor.Name]; ok {
				log.Printf("[%32s] Received new %s: %v\n", s.Name, sensor.Name, value)
				s.SetValue(sensor.Name, value)
			}
		}
	}
}

func (s *Sensor) addHistory(field string, value any) {
	s.mutex.Lock()
	s.Values[field].History = append(s.Values[field].History, SensorHistory{Time: time.Now(), Value: value})
	if len(s.Values[field].History) > 10 {
		s.Values[field].History = s.Values[field].History[len(s.Values[field].History)-10:]
	}
	s.mutex.Unlock()
}

func (s *Sensor) UpdateValue() (float64, bool) { return 0, false }

func (s *Sensor) ClearTrigger(trigger string) {
	if s.HasField(trigger) {
		s.SetValue(trigger, nil)
	}
}

func (s *Sensor) DisplaySince(field string) string {
	var idx int
	for i, sensor := range s.sensors {
		if sensor.Name == field {
			idx = i
		}
	}
	if s.Values[field] != nil {
		if fmt.Sprintf("%v", s.Values[field].Value) == fmt.Sprintf("%v", s.sensors[idx].ShowSince) {
			return "now"
		}
		for _, h := range s.Values[field].History {
			if fmt.Sprintf("%v", h.Value) == fmt.Sprintf("%v", s.sensors[idx].ShowSince) {
				return h.Time.Format("15:04")
			}
		}
	}
	return "--"
}
