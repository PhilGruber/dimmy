package devices

import (
	"encoding/json"
	core "github.com/PhilGruber/dimmy/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"time"
)

type Sensor struct {
	Dimmable

	TargetDevice      string
	TargetOnDuration  int
	TargetOffDuration int
	Timeout           int
	Active            bool
}

func MakeSensor(config core.DeviceConfig) Sensor {
	s := Sensor{}
	s.MqttTopic = config.Topic
	s.TargetDevice = *config.Options.Target

	s.Max = 100
	s.Min = 0

	s.TargetOnDuration = 3
	if config.Options.TargetOnDuration != nil {
		s.TargetOnDuration = *config.Options.TargetOnDuration
	}

	s.TargetOffDuration = 120
	if config.Options.TargetOffDuration != nil {
		s.TargetOffDuration = *config.Options.TargetOffDuration
	}

	s.Timeout = 10
	if config.Options.Timeout != nil {
		s.Timeout = *config.Options.Timeout
	}

	s.Hidden = false
	if config.Options != nil && config.Options.Hidden != nil {
		s.Hidden = *config.Options.Hidden
	}

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

func (s *Sensor) GetTimeoutRequest() (core.SwitchRequest, bool) {
	var request core.SwitchRequest

	if !s.Active {
		return request, false
	}

	if (s.LastChanged.Local().Add(time.Second * time.Duration(s.Timeout))).Before(time.Now()) {
		log.Println("Timeout")
		s.Active = false

		request.Device = s.TargetDevice
		request.Value = 0
		request.Duration = s.TargetOffDuration

		return request, true
	}

	return request, false

}

func (s *Sensor) GenerateRequest(cmd string) (core.SwitchRequest, bool) {
	var request core.SwitchRequest
	tt := time.Now()
	s.LastChanged = &tt
	s.Active = true
	request.Device = s.TargetDevice
	request.Value = s.Current
	request.Duration = s.TargetOnDuration
	return request, true
}

func (s *Sensor) PublishValue(mqtt mqtt.Client) {
}

func (s *Sensor) GetStateMqttTopic() string {
	return s.MqttState
}

func (s *Sensor) GetMessageHandler(channel chan core.SwitchRequest, sensor DeviceInterface) mqtt.MessageHandler {
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
