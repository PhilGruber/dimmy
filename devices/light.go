package devices

import (
	"encoding/json"
	"github.com/PhilGruber/dimmy/core"
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

func NewLight(config core.DeviceConfig) *Light {
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
		mqtt.Publish(l.MqttTopic, 0, false, strconv.Itoa(newVal))
	}
}

func (l *Light) GetMessageHandler(chan core.SwitchRequest, DeviceInterface) mqtt.MessageHandler {
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
		if l.GetTarget() != math.Round(l.GetCurrent()) {
			//			log.Printf("Ignoring Value %d from %s because light is moving", value, l.MqttState)
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

func (l *Light) GetTriggerValue(trigger string) interface{} {
	if trigger == "brightness" {
		return l.GetCurrent()
	}
	return nil
}
