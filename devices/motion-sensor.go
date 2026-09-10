package devices

import (
	"encoding/json"
	"log"
	"time"

	"github.com/PhilGruber/dimmy/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MotionSensor struct {
	Device

	Active bool
}

func (s *MotionSensor) UpdateValue() (float64, bool) {
	return 0, false
}

func NewMotionSensor(config core.DeviceConfig) *MotionSensor {
	s := MotionSensor{}
	s.setBaseConfig(config)
	s.MqttState = config.Topic

	s.Active = false
	s.Type = "motion-sensor"

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
			s.UpdateRules("motion", 1)
		}

	}
}
