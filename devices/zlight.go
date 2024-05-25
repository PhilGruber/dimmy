package devices

import (
	"encoding/json"
	core "github.com/PhilGruber/dimmy/core"
	"log"
	"math"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type ZLight struct {
	Light
}

func NewZLight(config core.DeviceConfig) *ZLight {
	d := makeZLight(config)
	return &d
}

func makeZLight(config core.DeviceConfig) ZLight {
	d := ZLight{}
	d.MqttTopic = config.Topic
	d.MqttState = config.Topic
	d.Current = 0
	d.Target = 0

	d.Hidden = false
	d.Min = 0
	d.Max = 100
	d.transition = false
	d.Type = "light"

	if config.Options != nil {
		if config.Options.Hidden != nil {
			d.Hidden = *config.Options.Hidden
		}
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
	return d
}

func (l *ZLight) PublishValue(mqtt mqtt.Client) {
	tt := time.Now()
	newVal := int(math.Round(l.Current * 2.5))
	var state string
	if newVal != l.LastSent {
		l.LastChanged = &tt
		l.LastSent = newVal

		if newVal > 0 {
			state = "ON"
		} else {
			state = "OFF"
		}
		brightness := int(math.Round(l.Current * 2.5))

		msg := core.Zigbee2MqttLightMessage{
			State:      state,
			Brightness: brightness,
		}

		if l.transition {
			msg.Transition = l.TransitionTime
		}

		s, _ := json.Marshal(msg)
		mqtt.Publish(l.MqttTopic+"/set", 0, false, s)
	}
}

func (l *ZLight) PollValue(mqtt mqtt.Client) {
	msg := core.Zigbee2MqttLightMessage{
		State:      "",
		Brightness: 0,
	}
	s, _ := json.Marshal(msg)
	log.Println("Polling " + l.MqttState)
	t := mqtt.Publish(l.MqttState+"/get", 0, false, s)
	if t.Wait() && t.Error() != nil {
		log.Println(t.Error())
	}
}

func (l *ZLight) GetMessageHandler(channel chan core.SwitchRequest, sw DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := mqttMessage.Payload()
		value, err := strconv.Atoi(string(payload))
		var state bool
		if err == nil {
			state = value > 0
		} else {
			var data core.Zigbee2MqttLightMessage
			err := json.Unmarshal(payload, &data)
			if err != nil {
				log.Println("Error: " + err.Error())
				return
			}
			value = data.Brightness
			state = data.State == "ON"
		}
		moving := l.Current != l.Target
		if moving {
			log.Printf("Ignoring value %d from %s because light is moving", value, l.MqttState)
			return
		}
		log.Printf("Received value %d from %s", value, l.MqttState)
		if state {
			l.Current = float64(value) / 2.5
		} else {
			l.Current = 0
		}
		if !moving {
			l.Target = l.Current
		}
	}
}
