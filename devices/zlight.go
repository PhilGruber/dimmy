package devices

import (
	"encoding/json"
	"github.com/PhilGruber/dimmy/core"
	"log"
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
	if newVal != l.LastSent {
		l.LastChanged = &tt
		l.LastSent = newVal

		if newVal > 0 {
			state = "ON"
		} else {
			state = "OFF"
		}

		msg := core.Zigbee2MqttLightMessage{
			State:      &state,
			Brightness: &newVal,
		}

		if l.transition {
			msg.Transition = &l.TransitionTime
		}

		s, _ := json.Marshal(msg)
		mqtt.Publish(l.MqttTopic+"/set", 0, false, s)
	}
}

func (l *ZLight) PollValue(mqtt mqtt.Client) {
	msg := core.Zigbee2MqttLightMessage{}
	s, _ := json.Marshal(msg)
	log.Printf("[%32s] Polling %s\n", l.GetName(), l.MqttState)
	t := mqtt.Publish(l.MqttState+"/get", 0, false, s)
	if t.Wait() && t.Error() != nil {
		log.Println(t.Error())
	}
}

func (l *ZLight) GetMessageHandler(channel chan core.SwitchRequest, sw DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := mqttMessage.Payload()
		var data core.Zigbee2MqttLightMessage
		err := json.Unmarshal(payload, &data)
		if err != nil {
			log.Println("Error: " + err.Error())
			return
		}
		if data.Brightness != nil {
			l.SetCurrent(l.ValueToPercentage(*data.Brightness))
		}
		if data.State != nil {
			if *data.State == "OFF" {
				l.SetCurrent(0)
			}
		}
		if data.Battery != nil {
			l.setBatteryLevel(data.Battery)
		}
		if data.LinkQuality != nil {
			l.setLinkQuality(data.LinkQuality)
		}
		if l.transition {
			l.setTarget(l.GetCurrent())
		}
	}
}
