package devices

import (
	"log"
	"math"
	"regexp"
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
	if config.Transformer == "" {
		config.Transformer = "tasmota"
	}
	d := Light{}
	d.Icon = "💡"
	d.setBaseConfig(config)

	d.persistentFields = []string{"brightness", "value"}

	var re = regexp.MustCompile("^cmnd/(.+)$")
	d.MqttState = re.ReplaceAllString(d.MqttTopic, "tele/$1/STATE")

	d.Target = 0

	d.Min = 0
	d.Max = 100

	d.Receivers = []string{"brightness", "duration"}

	if config.Options != nil {
		if config.Options.Min != nil {
			d.Min = *config.Options.Min
		}
		if config.Options.Max != nil {
			d.Max = *config.Options.Max
		}
	}

	if config.Transformer == "cbus" {
		d.Min = 0
		d.Max = 255
		d.transition = true
		d.MqttState = config.Topic
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
		brightness := value.(float64)
		l.setTarget(brightness)
		log.Printf("[%32s] Setting brightness to %f\n", l.GetName(), brightness)
	case "duration":
		duration := value.(int)
		log.Printf("[%32s] Setting duration to %d seconds\n", l.GetName(), duration)
	}
}

func (l *Light) PublishValue(mqtt mqtt.Client) {
	tt := time.Now()
	newVal := l.PercentageToValue(l.Current)
	if newVal != l.LastSent {
		l.LastChanged = &tt
		l.LastSent = newVal
		data := make(map[string]any)
		data["value"] = newVal
		if l.transition {
			data["transition"] = l.TransitionTime
		}
		topic, value, err := l.transformer.Encode(l.GetMqttTopic(), data)
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
		data, err := l.transformer.Decode(payload)
		if err != nil {
			log.Printf("[%32s] Error: %s\n", l.Name, err.Error())
			return
		}

		state := data["state"] == "ON"
		if !state {
			data["value"] = 0
		}
		if !l.transition && l.GetTarget() != math.Round(l.GetCurrent()) {
			return
		}
		value, ok := transform.IntValue(data["value"])
		if !ok {
			log.Printf("[%32s] Invalid light value: %v", l.Name, data["value"])
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
		Name:        name,
		Transformer: d.transformerName,
		Room:        d.Room,
		Type:        "light",
		Icon:        d.Icon,
		Topic:       d.MqttTopic,
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
