package devices

import (
	"log"
	"time"

	"github.com/PhilGruber/dimmy/core"
	"github.com/PhilGruber/dimmy/transform"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type ZLight struct {
	Light
	StupidHack bool
}

func NewZLight(config core.DeviceConfig) *ZLight {
	d := makeZLight(config)
	return &d
}

func makeZLight(config core.DeviceConfig) ZLight {
	d := ZLight{}
	d.Icon = "💡"
	d.setBaseConfig(config)
	d.MqttState = config.Topic

	d.Target = 0

	d.Min = 0
	d.Max = 254
	d.transition = false
	d.Type = "light"
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

		if config.Options.StupidHack {
			d.StupidHack = config.Options.StupidHack
		}
	}

	if config.Protocol == "cbus" {
		d.Min = 0
		d.Max = 255
		d.transition = true
	}

	d.Triggers = []string{"brightness"}
	d.persistentFields = []string{"brightness"}

	tt := time.Now()
	d.LastChanged = &tt
	d.init()
	return d
}

func (l *ZLight) PublishValue(mqtt mqtt.Client) {
	tt := time.Now()
	newVal := l.PercentageToValue(l.GetCurrent())
	var state string
	if newVal == l.LastSent {
		return
	}
	l.LastChanged = &tt
	l.LastSent = newVal

	if newVal > 0 || l.transition == true {
		state = "ON"
	} else {
		state = "OFF"
	}

	msg := make(map[string]any)
	msg["state"] = state
	msg["brightness"] = newVal
	if l.transition {
		msg["transition"] = l.TransitionTime
	}
	topic, s, err := l.protocol.Encode(l.GetMqttTopic(), msg)
	if err != nil {
		log.Println("Error: " + err.Error())
		return
	}

	mqtt.Publish(topic, 0, false, s)

}

func (l *ZLight) PollValue(mqtt mqtt.Client) {
	pollFields := make(map[string]any)
	pollFields["brightness"] = 0
	topic, s, err := l.protocol.PollMessage(l.MqttState, pollFields)
	if err != nil {
		log.Printf("[%32s] Error: %s", l.Name, err)
		return
	}
	if topic == "" {
		return
	}
	log.Printf("[%32s] Polling %s\n", l.GetName(), l.MqttState)
	t := mqtt.Publish(topic, 0, false, s)
	if t.Wait() && t.Error() != nil {
		log.Println(t.Error())
	}
}

func (l *ZLight) GetMessageHandler(channel chan core.SwitchRequest, sw DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := mqttMessage.Payload()
		data, err := l.protocol.Decode(payload)
		if err != nil {
			log.Printf("[%32s] Error: %s\n", l.Name, err.Error())
			return
		}
		if data["brightness"] != nil {
			brightness, ok := transform.IntValue(data["brightness"])
			if !ok {
				log.Printf("[%32s] Invalid brightness: %v", l.Name, data["brightness"])
				return
			}
			l.SetCurrent(l.ValueToPercentage(brightness))
		}
		if data["state"] != nil {
			if data["state"] == "OFF" {
				l.SetCurrent(0)
			}
		}
		l.parseDefaultValues(data)
		if l.transition {
			l.setTarget(l.GetCurrent())
		}
	}
}

func (d *ZLight) GetConfig(name string) core.DeviceConfig {
	config := core.DeviceConfig{
		Name:     name,
		Protocol: d.protocolName,
		Room:     d.Room,
		Type:     "zlight",
		Icon:     d.Icon,
		Topic:    d.MqttTopic,
		Options: &core.ConfigOptions{
			Transition: &d.transition,
			Hidden:     &d.Hidden,
		},
	}
	return config
}
