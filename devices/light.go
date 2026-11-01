package devices

import (
	"encoding/json"
	"log"
	"math"
	"time"

	"github.com/PhilGruber/dimmy/core"
	"github.com/PhilGruber/dimmy/transform"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Light struct {
	Dimmable
}

type lightStateMessage struct {
	Value int    `json:"Dimmer"`
	State string `json:"POWER"`
}

func NewLight(config core.DeviceConfig) *Light {
	if config.Protocol == "" {
		config.Protocol = "tasmota"
	}
	d := Light{}
	d.Icon = "💡"
	d.setBaseConfig(config)

	d.persistentFields = []string{"brightness"}

	d.MqttState = d.protocol.StateTopic(config.Topic)

	d.Target = 0

	d.transition = false

	d.Min = 0
	d.Max = 255
	if config.Protocol == "tasmota" {
		d.Max = 100
	}

	d.Receivers = []string{"brightness", "duration"}

	if config.Options != nil {
		if config.Options.Min != nil {
			d.Min = *config.Options.Min
		}
		if config.Options.Max != nil {
			d.Max = *config.Options.Max
		}
		if config.Options.Transition != nil {
			d.transition = *config.Options.Transition
		}
	}

	tt := time.Now()
	d.LastChanged = &tt
	d.Type = "light"
	d.Triggers = []string{"brightness"}

	d.init()

	return &d
}

func (l *Light) SetReceiverValue(key string, value interface{}) {
	switch key {
	case "brightness":
		brightness, ok := value.(float64)
		if !ok {
			log.Printf("[%32s] Invalid brightness: %v", l.Name, value)
			return
		}
		l.setTarget(brightness)
		log.Printf("[%32s] Setting brightness to %f\n", l.GetName(), brightness)
	case "duration":
		duration, ok := transform.IntValue(value)
		if !ok {
			log.Printf("[%32s] Invalid duration: %v", l.Name, value)
			return
		}
		log.Printf("[%32s] Setting duration to %d seconds\n", l.GetName(), duration)
	}
}

func (l *Light) PublishValue(mqtt mqtt.Client) {
	tt := time.Now()
	newVal := l.PercentageToValue(l.GetCurrent())
	if newVal != l.LastSent {
		l.LastChanged = &tt
		l.LastSent = newVal
		data := make(map[string]any)
		data["brightness"] = newVal
		if l.transition {
			data["transition"] = l.TransitionTime
		}
		topic, value, err := l.protocol.Encode(l.GetMqttTopic(), data)
		if err != nil {
			log.Println("Error: " + err.Error())
			return
		}
		mqtt.Publish(topic, 0, false, value)
	}
}

func (l *Light) GetMessageHandler(chan core.SwitchRequest, DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := mqttMessage.Payload()
		data, err := l.protocol.Decode(payload)
		if err != nil {
			log.Printf("[%32s] Error: %s\n", l.Name, err.Error())
			return
		}

		if data["state"] == "OFF" {
			data["brightness"] = 0
		}
		if !l.transition && l.GetTarget() != math.Round(l.GetCurrent()) {
			return
		}
		value, ok := transform.IntValue(data["brightness"])
		if !ok {
			log.Printf("[%32s] Invalid light value: %v", l.Name, data["brightness"])
			return
		}
		percentage := l.ValueToPercentage(value)
		l.setTarget(percentage)
		l.SetCurrent(percentage)
	}
}

func (l *Light) PercentageToValue(percentage float64) int {
	if percentage <= 1.0 {
		return l.GetMin() + int(math.Round(percentage))
	}
	return l.GetMin() + 1 + int(float64(l.GetMax()-l.GetMin()-1)*(percentage-1)/99)
}

func (l *Light) ValueToPercentage(value int) float64 {
	if value <= l.GetMin() {
		return 0
	}
	if value <= l.GetMin()+1 {
		return 1
	}
	if value >= l.GetMax() {
		return 100
	}
	return 1 + float64(value-l.GetMin()-1)*99/float64(l.GetMax()-l.GetMin()-1)
}

func (d *Light) GetConfig(name string) core.DeviceConfig {
	config := core.DeviceConfig{
		Name:     name,
		Protocol: d.protocolName,
		Room:     d.Room,
		Type:     "light",
		Icon:     d.Icon,
		Topic:    d.MqttTopic,
		Options: &core.ConfigOptions{
			Transition: &d.transition,
			Hidden:     &d.Hidden,
		},
	}
	return config
}

func (d *Light) GetMqttTopic() string {
	return d.MqttTopic
}

func (l *Light) PollValue(client mqtt.Client) {
	if err := transform.Poll(client, l.protocol, l.MqttState, map[string]any{"brightness": 0}); err != nil {
		log.Printf("[%32s] Poll failed: %v", l.Name, err)
	}
}

// Lights expose brightness while other device types retain their value field.
func (l *Light) MarshalJSON() ([]byte, error) {
	type lightData Light
	data, err := json.Marshal((*lightData)(l))
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return nil, err
	}
	fields["brightness"] = fields["value"]
	delete(fields, "value")
	return json.Marshal(fields)
}
