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

	onPressed  bool
	offPressed bool

	TargetDevice string
	Step         int
}

func makeSwitch(config core.DeviceConfig) Switch {
	s := Switch{}
	s.Name = config.Name
	s.MqttTopic = config.Topic
	s.MqttState = config.Topic
	s.TargetDevice = *config.Options.Target
	s.Type = "switch"
	s.Triggers = []string{"on", "off"}
	s.onPressed = false
	s.offPressed = false

	s.Hidden = true
	return s
}

func NewSwitch(config core.DeviceConfig) *Switch {
	p := makeSwitch(config)
	return &p
}

func (s *Switch) GenerateRequest(cmd string) (core.SwitchRequest, bool) {
	var request core.SwitchRequest
	request.Device = s.TargetDevice
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

		if data.Action == "on" {
			s.onPressed = true
		} else {
			s.offPressed = true
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

func (s *Switch) GetTriggerValue(key string) any {
	if key == "on" && s.onPressed {
		s.onPressed = false
		return 1
	}
	if key == "off" && s.offPressed {
		s.offPressed = false
		return 1
	}
	return 0
}
