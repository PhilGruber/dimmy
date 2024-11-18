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

type Plug struct {
	Device
	needsSending bool
	Min          int `json:"-"`
	Max          int `json:"-"`
}

type plugStateMessage struct {
	Value string `json:"POWER"`
}

func makePlug(config core.DeviceConfig) Plug {
	p := Plug{}
	p.Emoji = "🔌"
	p.setBaseConfig(config)

	var re = regexp.MustCompile("^cmnd/(.+)/POWER$")
	p.MqttState = re.ReplaceAllString(p.MqttTopic, "tele/$1/STATE")

	tt := time.Now()
	p.LastChanged = &tt
	p.Type = "plug"
	p.needsSending = false
	p.Min = 0
	p.Max = 1
	return p
}

func NewPlug(config core.DeviceConfig) *Plug {
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

func (p *Plug) GetMax() int {
	return p.Max
}

func (p *Plug) GetMin() int {
	return p.Min
}

func (p *Plug) ProcessRequest(request core.SwitchRequest) {
	val, _ := strconv.ParseFloat(request.Value, 64)
	if val != p.Current {
		p.Current = val
		if p.Current > 1 {
			p.Current = 1
		}
		if p.Current < 0 {
			p.Current = 0
		}
		p.needsSending = true
	}
}

func (p *Plug) GetMessageHandler(channel chan core.SwitchRequest, plug DeviceInterface) mqtt.MessageHandler {
	log.Println("Creating message handler for plug")
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		log.Println("Received message from plug")
		payload := mqttMessage.Payload()

		var message plugStateMessage
		err := json.Unmarshal(payload, &message)
		if err != nil {
			log.Println("Could parse status message from plug: " + err.Error())
		}

		log.Printf("Received state value %s from %s\n", message.Value, plug.GetMqttStateTopic())
		if message.Value == "ON" {
			p.setCurrent(1)
		} else {
			p.setCurrent(0)
		}
	}
}

func (p *Plug) PercentageToValue(percentage float64) int {
	if percentage <= 0.99 {
		return 0
	}
	return 1
}

func (p *Plug) ValueToPercentage(value int) float64 {
	if value == 0 {
		return 0
	}
	return 1
}

func (p *Plug) GetTriggerValue(key string) interface{} {
	return nil
}

func (p *Plug) SetReceiverValue(key string, value interface{}) {
	if key != "state" {
		return
	}
	p.ProcessRequest(core.SwitchRequest{Device: "", Value: value.(string)})
}
