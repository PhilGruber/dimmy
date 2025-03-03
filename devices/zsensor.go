package devices

import (
	"encoding/json"
	"github.com/PhilGruber/dimmy/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"time"
)

type ZSensor struct {
	Device
}

func MakeZSensor(config core.DeviceConfig) ZSensor {
	s := ZSensor{}
	s.setBaseConfig(config)
	s.MqttState = config.Topic
	s.Type = "sensor"

	return s
}

func NewZSensor(config core.DeviceConfig) *ZSensor {
	s := MakeZSensor(config)
	return &s
}

type ZSensorMessage struct {
	core.Zigbee2MqttMessage

	Occupancy bool `json:"occupancy"`
}

func (s *ZSensor) GetMessageHandler(channel chan core.SwitchRequest, sensor DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {

		payload := mqttMessage.Payload()

		log.Printf("%s", payload)

		var data ZSensorMessage
		err := json.Unmarshal(payload, &data)
		if err != nil {
			log.Println("Error: " + err.Error())
			return
		}

		val := "on"
		if data.Occupancy {
			val = "on"
			now := time.Now()
			s.LastChanged = &now
			s.SetCurrent(1)
		} else {
			val = "off"
			if s.GetCurrent() == 1 {
				now := time.Now()
				s.LastChanged = &now
			}
			s.SetCurrent(0)
		}
		s.setBatteryLevel(data.Battery)
		s.setLinkQuality(data.LinkQuality)
		log.Println(sensor.GetMqttTopic() + " is " + val)
	}
}

func (s *ZSensor) GetTriggerValue(trigger string) interface{} {
	if trigger == "sensor" {
		return s.GetCurrent()
	}
	return nil
}

func (s *ZSensor) ClearTrigger(trigger string) {
	if trigger == "sensor" {
		s.SetCurrent(-1)
	}
}

func (s *ZSensor) GenerateRequest(cmd string) (core.SwitchRequest, bool) {
	var request core.SwitchRequest
	return request, true
}

func (s *ZSensor) UpdateValue() (float64, bool) {
	return 0, false
}
