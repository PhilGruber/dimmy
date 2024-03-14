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

type Light struct {
	Dimmable
}

type lightStateMessage struct {
	Value float64 `json:"Dimmer"`
}

func makeLight(config map[string]string) Light {
	d := Light{}
	d.MqttTopic = config["topic"]
	var re = regexp.MustCompile("^cmnd/(.+)/dimmer$")
	d.MqttState = re.ReplaceAllString(d.MqttTopic, "tele/$1/STATE")

	if state, ok := config["state"]; ok {
		d.MqttState = state
	}
	d.Current = 0
	d.Target = 0
	min, ok := config["min"]
	if !ok {
		min = "0"
	}
	max, ok := config["max"]
	if !ok {
		max = "100"
	}

	d.Min, _ = strconv.Atoi(min)
	d.Max, _ = strconv.Atoi(max)

	d.Hidden = false
	if val, ok := config["hidden"]; ok {
		d.Hidden = val == "true"
	}

	tt := time.Now()
	d.LastChanged = &tt
	d.Type = "light"
	return d
}

func NewLight(config map[string]string) *Light {
	d := makeLight(config)
	return &d
}

func (l *Light) PublishValue(mqtt mqtt.Client) {
	tt := time.Now()
	newVal := int(math.Round(l.Current))
	if newVal != l.LastSent {
		l.LastChanged = &tt
		l.LastSent = newVal
		mqtt.Publish(l.MqttTopic, 0, false, strconv.Itoa(int(math.Round(l.Current))))
	}
}

func (l *Light) getMessageHandler(channel chan SwitchRequest, light DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := mqttMessage.Payload()
		var data lightStateMessage
		err := json.Unmarshal(payload, &data)
		if err != nil {
			log.Println("Error: " + err.Error())
			return
		}
		log.Printf("Received state value %.2f from %s\n", data.Value, light.getMqttStateTopic())
		if l.getTarget() == math.Round(l.getCurrent()) {
			l.setTarget(data.Value)
		}
		l.setCurrent(data.Value)

	}
}
