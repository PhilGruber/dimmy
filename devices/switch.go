package devices

import (
	"encoding/json"
	"github.com/PhilGruber/dimmy/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
)

type Switch struct {
	Device

	onPressed      bool
	offPressed     bool
	brightnessUp   bool
	brightnessDown bool
}

func makeSwitch(config core.DeviceConfig) Switch {
	s := Switch{}
	s.Name = config.Name
	s.MqttTopic = config.Topic
	s.MqttState = config.Topic

	s.Type = "switch"
	s.Triggers = []string{"button"}
	s.onPressed = false
	s.offPressed = false

	s.Hidden = true
	return s
}

func NewSwitch(config core.DeviceConfig) *Switch {
	p := makeSwitch(config)
	return &p
}

/*
func (s *Switch) GenerateRequest(cmd string) (core.SwitchRequest, bool) {
	var request core.SwitchRequest
	request.Device = s.TargetDevice
	request.Value = cmd
	request.Duration = 1
	return request, true
}
*/

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

		switch data.Action {
		case "on":
			s.onPressed = true
		case "off":
			s.offPressed = true
		case "brightness_move_up":
			s.brightnessUp = true
		case "brightness_move_down":
			s.brightnessDown = true
		case "brightness_stop":
			s.brightnessUp = false
			s.brightnessDown = false
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
	if key == "button" {
		if s.onPressed {
			return "on"
		}
		if s.offPressed {
			return "off"
		}
	}
	if key == "brightness" {
		if s.brightnessUp {
			return "up"
		}
		if s.brightnessDown {
			return "down"
		}
	}
	return nil
}

func (s *Switch) ClearTrigger(key string) {
	if key == "button" {
		s.onPressed = false
		s.offPressed = false
	}
}
