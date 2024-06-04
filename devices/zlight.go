package devices

import (
	"encoding/json"
	core "github.com/PhilGruber/dimmy/core"
	"log"
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
	d.Max = 254
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
	newVal := l.PercentageToValue(l.Current)
	var state string
	if newVal != l.LastSent {
		l.LastChanged = &tt
		l.LastSent = newVal

		if newVal > 0 {
			state = "ON"
		} else {
			state = "OFF"
		}

		msg := core.Zigbee2MqttLightMessage{
			State:      state,
			Brightness: newVal,
		}

		if l.transition {
			msg.Transition = &l.TransitionTime
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
		var on bool
		if err == nil {
			on = value > 0
		} else {
			var data core.Zigbee2MqttLightMessage
			err := json.Unmarshal(payload, &data)
			if err != nil {
				log.Println("Error: " + err.Error())
				return
			}
			value = data.Brightness
			on = data.State == "ON"
		}
		moving := l.Current != l.Target
		if moving {
			log.Printf("Ignoring value %d from %s because light is moving", value, l.MqttState)
			return
		}
		log.Printf("Received value %d from %s", value, l.MqttState)
		if on {
			l.Current = l.ValueToPercentage(value)
		} else {
			l.Current = 0
		}
		l.Target = l.Current
	}
}
