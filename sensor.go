package main

import (
    "log"
    "encoding/json"
    "time"
    mqtt "github.com/eclipse/paho.mqtt.golang"
)

type TuyaSensor struct {
    MqttTopic string `json:"-"`
    Target string
    TargetOn int
    TargetOff int
    TargetOnDuration int
    TargetOffDuration int
    Timeout int
    LastChanged *time.Time
    Active bool
}

func MakeTuyaSensor(config map[string]string) TuyaSensor {
    s := TuyaSensor{};
    s.MqttTopic = config["topic"]
    s.Target = config["target"]
    s.TargetOn = 100
    s.TargetOff = 0
    s.TargetOnDuration = 3
    s.TargetOffDuration = 120
    s.Timeout = 10
    s.Active = false
    return s
}

func NewTuyaSensor(config map[string]string) *TuyaSensor {
    s := MakeTuyaSensor(config)
    return &s
}

type TuyaSensorMessage struct {
    Data string
    Cmnd int
    CmndData string
}

type TuyaSensorMessageWrapper struct {
    TuyaReceived TuyaSensorMessage
}

func TuyaSensorMessageHandler(channel chan SwitchRequest, sensor *TuyaSensor) mqtt.MessageHandler {
    return func (client mqtt.Client, mqttMessage mqtt.Message) {
        payload := mqttMessage.Payload()

        log.Println("[" + sensor.MqttTopic + "] Payload: " + string(payload[:]))

        var data TuyaSensorMessageWrapper
        err := json.Unmarshal(payload, &data)
        if err != nil {
            log.Println("Error: ", err)
            return
        }

        message := data.TuyaReceived

        if message.Cmnd == 5 {
            log.Println("Received Command " + message.CmndData)
            tt := time.Now()
            sensor.LastChanged = &tt
            sensor.Active = true
            var request SwitchRequest
            request.Device   = sensor.Target
            request.Value    = sensor.TargetOn
            request.Duration = sensor.TargetOnDuration
            channel <- request
        }
    }
}
