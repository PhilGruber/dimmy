package devices

import (
	"encoding/json"
	core "github.com/PhilGruber/dimmy/core"
	"log"
	"strconv"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Switch struct {
	Device

	TarGetDevice string
	Step         int
}

func makeSwitch(config map[string]string) Switch {
	s := Switch{}
	s.MqttTopic = config["topic"]
	s.MqttState = config["topic"]
	s.TarGetDevice = config["tarGet"]
	s.Type = "switch"

	s.Hidden = true
	return s
}

func NewSwitch(config map[string]string) *Switch {
	p := makeSwitch(config)
	return &p
}

func (s *Switch) GenerateRequest(cmd string) (core.SwitchRequest, bool) {
	var request core.SwitchRequest
	request.Device = s.TarGetDevice
	request.Value, _ = strconv.ParseFloat(cmd, 64)
	request.Duration = 1
	return request, true
}

type SwitchMessage struct {
	core.Zigbee2MqttMessage

	Action string `json:"action"`
}

func (s *Switch) GetMessageHandler(channel chan core.SwitchRequest, sw DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {

		payload := mqttMessage.Payload()

		if sw.GetCurrent() == 0 {
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

		request, ok := sw.GenerateRequest(strconv.Itoa(val))
		if ok {
			channel <- request
		}

	}
}

func (s *Switch) UpdateValue() (float64, bool) {
	return 0, false
}

func (s *Switch) GetMax() int {
	return 1
}

func (s *Switch) GetMin() int {
	return 0
}

func (s *Switch) GetCurrent() float64 {
	return 1
}

func (s *Switch) setCurrent(float64) {
}

func (s *Switch) ProcessRequest(request core.SwitchRequest) {
}
