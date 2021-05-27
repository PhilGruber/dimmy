package main

import (
    "strconv"
    "encoding/json"
    "log"

    mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Switch struct {
    Device

    TargetDevice string
    Step int
}

func makeSwitch(config map[string]string) Switch {
    s := Switch{};
    s.MqttTopic = config["topic"]
    s.TargetDevice = config["target"]
    s.Type = "switch"
    return s
}

func NewSwitch(config map[string]string) *Switch {
    p := makeSwitch(config)
    return &p
}

func (s *Switch) generateRequest(cmd string) (SwitchRequest, bool) {
    var request SwitchRequest
    request.Device   = s.TargetDevice
    request.Value, _ = strconv.Atoi(cmd)
    request.Duration = 1
    return request, true
}

type SwitchMessageUpdate struct {
    state string
}

type SwitchMessage struct {
    Action string `json:"action"`
    Battery int `json:"battery"`
    Linkquality int `json:"linkquality"`
    UpdateAvailable bool `json:"`
    Update SwitchMessageUpdate `json:"update"`
}

func SwitchMessageHandler(channel chan SwitchRequest, s DeviceInterface) mqtt.MessageHandler {
    return func (client mqtt.Client, mqttMessage mqtt.Message) {

        payload := mqttMessage.Payload()

        if s.getCurrent() == 0 {
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

        request, ok := s.generateRequest(strconv.Itoa(val))
        if (ok) {
            channel <- request
        }


    }
}

func (s *Switch) UpdateValue() (float64, bool) {
    return 0, false
}

func (s *Switch) getMax() (int) {
    return 1
}

func (s *Switch) getMin() (int) {
    return 0
}

func (s *Switch) getCurrent() (float64) {
    return 1
}

func (s *Switch) setCurrent(float64) {
}

func (s *Switch) processRequest(request SwitchRequest) {
}
