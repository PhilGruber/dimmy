package devices

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/PhilGruber/dimmy/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
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

type GenericDevice struct {
	Device

	sensors    []core.Sensor
	controls   []core.Control
	Values     map[string]*SensorValue `json:"Values"`
	hasHistory bool
	valueMutex *sync.RWMutex
}

func NewDevice(config core.DeviceConfig) *GenericDevice {
	s := GenericDevice{}
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
		if s.sensors[i].Icon == "" {
			s.sensors[i].Icon = getIcon(s.sensors[i].Name)
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

func (d *GenericDevice) GetFields() []string {
	fields := make([]string, 0, len(d.sensors))
	for _, sensor := range d.sensors {
		fields = append(fields, sensor.Name)
	}
	return fields
}

func (d *GenericDevice) GetSensors() []core.Sensor {
	return d.sensors
}

func (d *GenericDevice) HasField(field string) bool {
	for _, s := range d.sensors {
		if s.Name == field {
			return true
		}
	}
	return false
}

func (d *GenericDevice) SetValue(field string, value any) {
	d.valueMutex.Lock()
	d.Values[field].Value = value
	d.Values[field].LastChanged = time.Now()

	for _, sensor := range d.sensors {
		if sensor.ShowSince != nil && sensor.Name == field {
			if fmt.Sprintf("%v", value) == fmt.Sprintf("%v", *sensor.ShowSince) {
				now := time.Now()
				d.Values[field].Since = &now
			}
		}
	}

	d.valueMutex.Unlock()
	if d.hasHistory {
		d.addHistory(field, value)
	}

	d.UpdateRules(field, value)
}

func (d *GenericDevice) GetValue(field string) any {
	d.valueMutex.RLock()
	defer d.valueMutex.RUnlock()
	return d.Values[field].Value
}

func (d *GenericDevice) GetMessageHandler(_ chan core.SwitchRequest, _ DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := mqttMessage.Payload()
		var data map[string]any
		err := json.Unmarshal(payload, &data)
		if err != nil {
			log.Printf("[%32s] Error: %d\n", d.Name, err.Error())
			return
		}

		d.parseDefaultValues(data)

		for _, sensor := range d.sensors {
			if value, ok := data[sensor.Name]; ok {
				log.Printf("[%32s] Received new %d: %v\n", d.Name, sensor.Name, value)
				d.SetValue(sensor.Name, value)
			}
		}
	}
}

func (d *GenericDevice) addHistory(field string, value any) {
	d.mutex.Lock()
	d.Values[field].History = append(d.Values[field].History, SensorHistory{Time: time.Now(), Value: value})
	if len(d.Values[field].History) > 10 {
		d.Values[field].History = d.Values[field].History[len(d.Values[field].History)-10:]
	}
	d.mutex.Unlock()
}

func (d *GenericDevice) UpdateValue() (float64, bool) {
	for _, control := range d.controls {
		if control.NeedsSending {
			return 0, true
		}
	}
	return 0, false
}

func (d *GenericDevice) PublishValue(client mqtt.Client) {
	values := make(map[string]any)
	for _, control := range d.controls {
		if control.NeedsSending {
			values[control.Name] = control.Value
			control.NeedsSending = false
		}
	}
	if len(values) == 0 {
		return
	}

	message, err := json.Marshal(values)
	if err != nil {
		log.Printf("[%32s] Error: %s\n", d.Name, err.Error())
		return
	}
	t := client.Publish(d.MqttTopic+"/set", 0, false, message)
	if t.Wait() && t.Error() != nil {
		log.Println(t.Error())
	}
}

func (d *GenericDevice) PollValue(client mqtt.Client) {
	values := make(map[string]any)
	for _, control := range d.controls {
		values[control.Name] = ""
	}
	if len(values) == 0 {
		return
	}

	s, _ := json.Marshal(values)
	log.Printf("[%32s] Polling value for %s\n", d.GetName(), d.MqttState)
	t := client.Publish(d.MqttState+"/get", 0, false, s)
	if t.Wait() && t.Error() != nil {
		log.Println(t.Error())
	}
}

func (d *GenericDevice) DisplaySince(field string) string {
	var idx int
	for i, sensor := range d.sensors {
		if sensor.Name == field {
			idx = i
		}
	}
	if d.Values[field] != nil {
		if fmt.Sprintf("%v", d.Values[field].Value) == fmt.Sprintf("%v", d.sensors[idx].ShowSince) {
			return "now"
		}
		for _, h := range d.Values[field].History {
			if fmt.Sprintf("%v", h.Value) == fmt.Sprintf("%v", d.sensors[idx].ShowSince) {
				return h.Time.Format("15:04")
			}
		}
	}
	return "--"
}

func getIcon(deviceType string) string {
	switch deviceType {
	case "humidity":
		return "💧"
	case "temperature":
		return "🌡️"
	case "illuminance":
		return "🔆"
	case "button":
		return "🔘"
	case "action":
		return "⚙️"
	case "presence":
		return "🧍"
	case "vibration":
		return "📳"
	case "contact":
		return "🚪"
	case "battery":
		return "🔋"
	case "power":
		return "⚡"
	case "light":
		return "💡"
	}
	return ""
}
