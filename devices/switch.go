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

func NewSwitch(config core.DeviceConfig) *Switch {
	s := Switch{}
	s.setBaseConfig(config)
	s.MqttState = config.Topic

	s.Type = "switch"
	s.Triggers = []string{"button"}
	s.onPressed = false
	s.offPressed = false

	s.Hidden = true

	return &s
}

type SwitchMessage struct {
	core.Zigbee2MqttMessage
	Action string `json:"action"`
}

func (s *Switch) GetMessageHandler(channel chan core.SwitchRequest, sw DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := mqttMessage.Payload()

		var data SwitchMessage
		err := json.Unmarshal(payload, &data)
		if err != nil {
			log.Println("Error: " + err.Error())
			return
		}

		log.Printf("[%32s] Button pressed (%s)\n", s.GetName(), data.Action)

		switch data.Action {
		case "on":
			s.onPressed = true
			s.UpdateRules("button", "on")
		case "off":
			s.offPressed = true
			s.UpdateRules("button", "off")
		case "brightness_move_up":
			s.brightnessUp = true
			s.UpdateRules("brightness", "up")
		case "brightness_move_down":
			s.brightnessDown = true
			s.UpdateRules("brightness", "down")
		case "brightness_stop":
			s.brightnessUp = false
			s.brightnessDown = false
			s.UpdateRules("brightness", "stop")
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

func (s *Switch) setCurrent(float64) {
}

func (s *Switch) ProcessRequest(request core.SwitchRequest) {
}

func (s *Switch) ClearTrigger(key string) {
	if key == "button" {
		s.onPressed = false
		s.offPressed = false
	}
}
