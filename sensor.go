package main

import (
    "log"
    "encoding/json"
    "time"
    "strconv"
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
    Value int
}

func MakeTuyaSensor(config map[string]string) TuyaSensor {
    s := TuyaSensor{};
    s.MqttTopic = config["topic"]
    s.Target = config["target"]

    var val string
    var ok bool

    if val, ok = config["TargetOn"]; !ok {
        val = "100"
    }
    s.TargetOn, _ = strconv.Atoi(val)

    if val, ok = config["TargetOff"]; !ok {
        val = "0"
    }
    s.TargetOff, _ = strconv.Atoi(val)

    if val, ok = config["TargetOnDuration"]; !ok {
        val = "3"
    }
    s.TargetOnDuration, _ = strconv.Atoi(val)

    if val, ok = config["TargetOffDuration"]; !ok {
        val = "120"
    }
    s.TargetOffDuration, _ = strconv.Atoi(val)

    if val, ok = config["Timeout"]; !ok {
        val = "10"
    }
    s.Timeout, _ = strconv.Atoi(val)

    s.Active = false
    s.Value = 1
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

        if sensor.Value == 0 {
            return
        }

        var data TuyaSensorMessageWrapper
        err := json.Unmarshal(payload, &data)
        if err != nil {
            log.Println("Error: ", err)
            return
        }

        message := data.TuyaReceived

        if message.Cmnd == 5 || message.Cmnd == 2 {
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
