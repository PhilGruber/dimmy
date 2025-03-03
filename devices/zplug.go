package devices

import (
	"encoding/json"
	"github.com/PhilGruber/dimmy/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
)

type ZPlug struct {
	Plug
}

type zplugStateMessage struct {
	core.Zigbee2MqttMessage
	State     string  `json:"state"`
	ChildLock string  `json:"child_lock"`
	Current   float64 `json:"current"`
	Energy    float64 `json:"energy"`
}

func NewZPlug(config core.DeviceConfig) *ZPlug {
	p := MakeZPlug(config)
	return &p
}

func MakeZPlug(config core.DeviceConfig) ZPlug {
	p := ZPlug{}
	p.Emoji = "🔌"
	p.setBaseConfig(config)

	p.MqttState = config.Topic
	p.Type = "plug"

	p.needsSending = false
	p.Min = 0
	p.Max = 1

	return p
}

func (p *ZPlug) GetMessageHandler(channel chan core.SwitchRequest, plug DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := mqttMessage.Payload()

		var message zplugStateMessage
		err := json.Unmarshal(payload, &message)
		if err != nil {
			log.Println("Could parse status message from plug: " + err.Error())
		}
		if message.LinkQuality != nil {
			p.setLinkQuality(message.LinkQuality)
		}

		if message.State != "" {
			log.Printf("[%32s] Received state Value %s\n", p.GetName(), message.State)
			if message.State == "ON" {
				p.SetCurrent(1)
			} else {
				p.SetCurrent(0)
			}
		}
	}
}

func (p *ZPlug) GetState() string {
	if p.Current == 1 {
		return "ON"
	}
	return "OFF"
}

func (p *ZPlug) PublishValue(client mqtt.Client) {
	log.Println("Publishing plug value")
	message := core.Zigbee2MqttMessageUpdate{
		State: p.GetState(),
	}
	payload, err := json.Marshal(message)
	if err != nil {
		log.Println("Error: " + err.Error())
		return
	}
	client.Publish(p.GetMqttStateTopic()+"/set", 0, false, payload)
	p.needsSending = false
}
