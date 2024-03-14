package main

import (
	"encoding/json"
	"log"
	"strconv"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Switch struct {
	Device

	TargetDevice string
	Step         int
}

func makeSwitch(config map[string]string) Switch {
	s := Switch{}
	s.MqttTopic = config["topic"]
	s.MqttState = config["topic"]
	s.TargetDevice = config["target"]
	s.Type = "switch"

	s.Hidden = true
	return s
}

func NewSwitch(config map[string]string) *Switch {
	p := makeSwitch(config)
	return &p
}

func (s *Switch) generateRequest(cmd string) (SwitchRequest, bool) {
	var request SwitchRequest
	request.Device = s.TargetDevice
	request.Value, _ = strconv.ParseFloat(cmd, 64)
	request.Duration = 1
	return request, true
}

type SwitchMessage struct {
	Zigbee2MqttMessage

	Action string `json:"action"`
}

func (s *Switch) getMessageHandler(channel chan SwitchRequest, sw DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {

		payload := mqttMessage.Payload()

		if sw.getCurrent() == 0 {
			return
		}

		var data SwitchMessage
		err := json.Unmarshal(payload, &data)
		if err != nil {
			log.Println("Error: " + err.Error())
			return
		}

		log.Printf("Button pressed (%s)", data.Action)
		val := 0

		if data.Action == "on" {
			val = 100
		}

		log.Printf("Setting device to %d", val)

		request, ok := sw.generateRequest(strconv.Itoa(val))
		if ok {
			channel <- request
		}

	}
}

func (s *Switch) UpdateValue() (float64, bool) {
	return 0, false
}

func (s *Switch) getMax() int {
	return 1
}

func (s *Switch) getMin() int {
	return 0
}

func (s *Switch) getCurrent() float64 {
	return 1
}

func (s *Switch) setCurrent(float64) {
}

func (s *Switch) processRequest(request SwitchRequest) {
}
