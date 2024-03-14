package main

import (
	"encoding/json"
	"log"
	"math"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type ZLight struct {
	Light
}

func NewZLight(config map[string]string) *ZLight {
	d := makeZLight(config)
	return &d
}

func makeZLight(config map[string]string) ZLight {
	d := ZLight{}
	d.MqttTopic = config["topic"]
	d.MqttState = config["topic"]
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
		d.Hidden = (val == "true")
	}

	tt := time.Now()
	d.LastChanged = &tt
	d.Type = "zlight"
	return d
}

func (l *ZLight) PublishValue(mqtt mqtt.Client) {
	tt := time.Now()
	newVal := int(math.Round(l.Current))
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

		msg := Zigbee2MqttLightMessage{
			State:      state,
			Brightness: brightness,
		}
		s, _ := json.Marshal(msg)
		mqtt.Publish(l.MqttTopic+"/set", 0, false, s)
	}
}

func (l *ZLight) PollValue(mqtt mqtt.Client) {
	msg := Zigbee2MqttLightMessage{
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

func (l *ZLight) getMessageHandler(channel chan SwitchRequest, sw DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := mqttMessage.Payload()
		var data Zigbee2MqttLightMessage
		err := json.Unmarshal(payload, &data)
		if err != nil {
			log.Println("Error: " + err.Error())
			return
		}
		moving := l.Current != l.Target
		log.Printf("Received value %d from %s", data.Brightness, l.MqttState)
		if data.State == "ON" {
			l.Current = float64(data.Brightness) / 2.5
		} else {
			l.Current = 0
		}
		if !moving {
			l.Target = l.Current
		}
	}
}
