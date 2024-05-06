package main

import (
	"encoding/json"
	"log"
	"math"
	"regexp"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Plug struct {
	Device
	needsSending bool
	Min          int `json:"-"`
	Max          int `json:"-"`
}

type plugStateMessage struct {
	Value string `json:"POWER"`
}

func makePlug(config map[string]string) Plug {
	p := Plug{}
	p.MqttTopic = config["topic"]
	var re = regexp.MustCompile("^cmnd/(.+)/POWER$")
	p.MqttState = re.ReplaceAllString(p.MqttTopic, "tele/$1/STATE")
	p.Current = 0

	p.Hidden = false
	if val, ok := config["hidden"]; ok {
		p.Hidden = (val == "true")
	}

	tt := time.Now()
	p.LastChanged = &tt
	p.Type = "plug"
	p.needsSending = false
	p.Min = 0
	p.Max = 1
	return p
}

func NewPlug(config map[string]string) *Plug {
	p := makePlug(config)
	return &p
}

func (p *Plug) PublishValue(mqtt mqtt.Client) {
	mqtt.Publish(p.MqttTopic, 0, false, strconv.Itoa(int(math.Round(p.Current))))
	p.needsSending = false
}

func (p *Plug) UpdateValue() (float64, bool) {
	if p.needsSending {
		return p.Current, true
	}
	return 0, false
}

func (p *Plug) getMax() int {
	return p.Max
}

func (p *Plug) getMin() int {
	return p.Min
}

func (p *Plug) processRequest(request SwitchRequest) {
	if request.Value != p.Current {
		p.Current = request.Value
		if p.Current > 1 {
			p.Current = 1
		}
		if p.Current < 0 {
			p.Current = 0
		}
		p.needsSending = true
	}
}

func (p *Plug) getMessageHandler(channel chan SwitchRequest, plug DeviceInterface) mqtt.MessageHandler {
	log.Println("Creating message handler for plug")
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		log.Println("Received message from plug")
		payload := mqttMessage.Payload()

		var message plugStateMessage
		err := json.Unmarshal(payload, &message)
		if err != nil {
			log.Println("Could parse status message from plug: " + err.Error())
		}

		log.Printf("Received state value %s from %s\n", message.Value, plug.getMqttStateTopic())
		if message.Value == "ON" {
			p.setCurrent(1)
		} else {
			p.setCurrent(0)
		}
	}
}
