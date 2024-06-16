package devices

import (
	"encoding/json"
	core "github.com/PhilGruber/dimmy/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"math"
)

type ZSensor struct {
	Dimmable

	TargetDevice      string
	TargetOnDuration  int
	TargetOffDuration int
	Timeout           int
}

func MakeZSensor(config core.DeviceConfig) ZSensor {
	s := ZSensor{}
	s.Name = config.Name
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

	s.Current = 0
	s.Target = 0
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

func (s *ZSensor) GetTimeoutRequest() (core.SwitchRequest, bool) {
	var request core.SwitchRequest
	return request, false
}

func (s *ZSensor) GetMessageHandler(channel chan core.SwitchRequest, sensor DeviceInterface) mqtt.MessageHandler {
	log.Println("Subscribing to " + sensor.GetMqttTopic())
	return func(client mqtt.Client, mqttMessage mqtt.Message) {

		payload := mqttMessage.Payload()

		log.Printf("%s", payload)

		if sensor.GetCurrent() == 0 {
			return
		}

		var data ZSensorMessage
		err := json.Unmarshal(payload, &data)
		if err != nil {
			log.Println("Error: " + err.Error())
			return
		}

		val := "on"
		if data.Occupancy {
			val = "on"
		} else {
			val = "off"
		}
		log.Println(sensor.GetMqttTopic() + " is " + val)
		request, ok := sensor.GenerateRequest(val)

		if ok {
			channel <- request
		}

	}
}

func (s *ZSensor) GenerateRequest(cmd string) (core.SwitchRequest, bool) {
	var request core.SwitchRequest
	request.Device = s.TargetDevice
	if cmd == "on" {
		request.Value = math.Round(s.Current)
		request.Duration = s.TargetOnDuration
	} else {
		request.Value = 0
		request.Duration = s.TargetOffDuration
	}
	return request, true
}
