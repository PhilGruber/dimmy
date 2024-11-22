package devices

import (
	"encoding/json"
	"github.com/PhilGruber/dimmy/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"time"
)

type Sensor struct {
	Dimmable

	Active bool
}

func MakeSensor(config core.DeviceConfig) Sensor {
	s := Sensor{}
	s.setBaseConfig(config)

	s.Max = 100
	s.Min = 0

	s.Active = false
	s.Current = 0
	s.Target = 0
	s.Type = "sensor"
	return s
}

func NewSensor(config core.DeviceConfig) *Sensor {
	s := MakeSensor(config)
	return &s
}

type SensorMessage struct {
	Data     string
	Cmnd     int
	CmndData string
}

type SensorMessageWrapper struct {
	TuyaReceived SensorMessage
}

func (s *Sensor) PublishValue(mqtt.Client) {
}

func (s *Sensor) GetStateMqttTopic() string {
	return s.MqttState
}

func (s *Sensor) GetMessageHandler(channel chan core.SwitchRequest, _ DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {

		payload := mqttMessage.Payload()

		if s.GetCurrent() == 0 {
			return
		}

		var data SensorMessageWrapper
		err := json.Unmarshal(payload, &data)
		if err != nil {
			log.Println("Error: " + err.Error())
			return
		}

		message := data.TuyaReceived

		if message.Cmnd == 5 || message.Cmnd == 2 {
			log.Printf("Motion detected (%d)", message.Cmnd)
			request, ok := s.GenerateRequest(message.CmndData)
			if ok {
				channel <- request
			}
		}

	}
}

func (s *Sensor) GetTriggerValue(key string) interface{} {
	if key == "noMotion" {
		return time.Now().Unix() - s.LastChanged.Unix()
	}
	return nil
}
