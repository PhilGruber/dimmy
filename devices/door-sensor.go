package devices

import (
	"encoding/json"
	"github.com/PhilGruber/dimmy/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
)

type DoorSensor struct {
	Device

	state        string
	triggerState string
}

type DoorSensorMessage struct {
	core.Zigbee2MqttMessage
	Contact string `json:"contact"`
}

func MakeDoorSensor(config core.DeviceConfig) DoorSensor {
	s := DoorSensor{}
	s.setBaseConfig(config)

	s.Type = "door-sensor"
	s.Triggers = []string{"sensor"}
	s.state = ""
	s.triggerState = ""

	return s
}

func NewDoorSensor(config core.DeviceConfig) *DoorSensor {
	s := MakeDoorSensor(config)
	return &s
}

func (s *DoorSensor) GetMessageHandler(channel chan core.SwitchRequest, sw DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := mqttMessage.Payload()
		var data DoorSensorMessage
		err := json.Unmarshal(payload, &data)
		if err != nil {
			log.Println("Error: " + err.Error())
			return
		}
		if data.Contact == "true" {
			s.state = "open"
		} else {
			s.state = "closed"
		}
		log.Printf("Door is %s\n", s.state)

		s.triggerState = s.state
	}
}

func (s *DoorSensor) GetTriggerValue(trigger string) interface{} {
	if trigger == "sensor" {
		return s.triggerState
	}
	return nil
}

func (s *DoorSensor) ClearTrigger(trigger string) {
	if trigger == "sensor" {
		s.triggerState = ""
	}
}

func (s *DoorSensor) UpdateValue() (float64, bool) {
	return 0, false
}

func (s *DoorSensor) GetMax() int {
	return 1
}
func (s *DoorSensor) GetMin() int {
	return 0
}
func (s *DoorSensor) GetCurrent() float64 {
	return 1
}
func (s *DoorSensor) ProcessRequest(request core.SwitchRequest) {
}
