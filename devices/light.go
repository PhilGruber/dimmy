package devices

import (
	"encoding/json"
	core "github.com/PhilGruber/dimmy/core"
	"log"
	"math"
	"regexp"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Light struct {
	Dimmable
}

type lightStateMessage struct {
	Value int    `json:"Dimmer"`
	State string `json:"POWER"`
}

func makeLight(config core.DeviceConfig) Light {
	d := Light{}
	d.Emoji = "💡"
	d.setBaseConfig(config)

	var re = regexp.MustCompile("^cmnd/(.+)/dimmer$")
	d.MqttState = re.ReplaceAllString(d.MqttTopic, "tele/$1/STATE")

	d.Current = 0
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

	tt := time.Now()
	d.LastChanged = &tt
	d.Type = "light"
	return d
}

func NewLight(config core.DeviceConfig) *Light {
	d := makeLight(config)
	return &d
}

func (l *Light) SetReceiverValue(key string, value interface{}) {
	switch key {
	case "brightness":
		brightness := value.(float64)
		l.setTarget(brightness)
		log.Printf("Setting brightness to %f\n", brightness)
	case "duration":
		duration := value.(int)
		log.Printf("Setting duration to %d seconds\n", duration)
	}
}

func (l *Light) GetTriggerValue(key string) interface{} {
	return nil
}

func (l *Light) PublishValue(mqtt mqtt.Client) {
	tt := time.Now()
	newVal := l.PercentageToValue(l.Current)
	if newVal != l.LastSent {
		l.LastChanged = &tt
		l.LastSent = newVal
		mqtt.Publish(l.MqttTopic, 0, false, strconv.Itoa(newVal))
	}
}

func (l *Light) GetMessageHandler(channel chan core.SwitchRequest, light DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := mqttMessage.Payload()
		value, err := strconv.Atoi(string(payload))
		state := value > 0
		if err != nil {
			var data lightStateMessage
			err := json.Unmarshal(payload, &data)
			if err != nil {
				log.Println("Error: " + err.Error())
				return
			}
			value = data.Value
			state = data.State == "ON"
		}
		if !state {
			value = 0
		}
		log.Printf("Received state value %d from %s\n", value, light.GetMqttStateTopic())
		if l.GetTarget() != math.Round(l.GetCurrent()) {
			log.Printf("Ignoring value %d from %s because light is moving", value, l.MqttState)
			return
		}
		percentage := l.ValueToPercentage(value)
		l.setTarget(percentage)
		l.setCurrent(percentage)

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
