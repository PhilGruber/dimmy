package devices

import (
	"encoding/json"
	"github.com/PhilGruber/dimmy/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"time"
)

type MotionSensor struct {
	Dimmable

	Active bool
}

func MakeMotionSensor(config core.DeviceConfig) MotionSensor {
	s := MotionSensor{}
	s.setBaseConfig(config)
	s.MqttState = config.Topic

	s.Max = 100
	s.Min = 0

	s.Active = false

	s.Type = "motion-sensor"
	return s
}

func NewMotionSensor(config core.DeviceConfig) *MotionSensor {
	s := MakeMotionSensor(config)
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

func (s *MotionSensor) PublishValue(mqtt.Client) {
}

func (s *MotionSensor) GetMessageHandler(_ chan core.SwitchRequest, _ DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {

		payload := mqttMessage.Payload()

		var data SensorMessageWrapper
		err := json.Unmarshal(payload, &data)
		if err != nil {
			log.Println("Error: " + err.Error())
			return
		}

		message := data.TuyaReceived

		if message.Cmnd == 5 || message.Cmnd == 2 {
			s.Active = true
			now := time.Now()
			s.LastChanged = &now
			log.Printf("Motion detected (%d)", message.Cmnd)
		}

	}
}

func (s *MotionSensor) GetTriggerValue(key string) interface{} {
	if key == "noMotion" {
		if s.LastChanged != nil {
			return time.Now().Unix() - s.LastChanged.Unix()
		}
		return 0
	}
	if key == "motion" {
		if s.Active {
			s.Active = false
			return 1
		}
	}
	return nil
}
