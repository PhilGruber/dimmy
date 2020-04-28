package main

import (
    "log"
    "encoding/json"
    "time"
    "strconv"
    "math"
    mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Sensor struct {
    Device

    TargetDevice string
    TargetOnDuration int
    TargetOffDuration int
    Timeout int
    Active bool
}

func MakeSensor(config map[string]string) Sensor {
    s := Sensor{};
    s.MqttTopic = config["topic"]
    s.TargetDevice = config["target"]

    s.Max = 100
    s.Min = 0

    var val string
    var ok bool

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
    s.Current = 0
    s.Target = 0
    s.Type = "sensor"
    return s
}

func NewSensor(config map[string]string) *Sensor {
    s := MakeSensor(config)
    return &s
}

type SensorMessage struct {
    Data string
    Cmnd int
    CmndData string
}

type SensorMessageWrapper struct {
    TuyaReceived SensorMessage
}

func (s *Sensor) getTimeoutRequest() (SwitchRequest, bool) {
    var request SwitchRequest

    if !s.Active {
        return request, false
    }

    if (s.LastChanged.Local().Add(time.Second * time.Duration(s.Timeout))).Before(time.Now()) {
        log.Println("Timeout")
        s.Active = false

        request.Device   = s.TargetDevice
        request.Value    = 0
        request.Duration = s.TargetOffDuration

        return request, true
    }

    return request, false

}

func (s *Sensor) generateMotionRequest(cmd string) SwitchRequest {
    var request SwitchRequest
    tt := time.Now()
    s.LastChanged = &tt
    s.Active = true
    request.Device   = s.TargetDevice
    request.Value    = int(math.Round(s.Current))
    request.Duration = s.TargetOnDuration
    return request
}

func SensorMessageHandler(channel chan SwitchRequest, sensor DeviceInterface) mqtt.MessageHandler {
    return func (client mqtt.Client, mqttMessage mqtt.Message) {

        payload := mqttMessage.Payload()

        log.Println("Sensor Message received")

        if sensor.getCurrent() == 0 {
            return
        }

        var data SensorMessageWrapper
        err := json.Unmarshal(payload, &data)
        if err != nil {
            log.Println("Error: ", err)
            return
        }

        message := data.TuyaReceived

        if message.Cmnd == 5 || message.Cmnd == 2 {
            log.Printf("Motion detected (%d)", message.Cmnd)
            request := sensor.generateMotionRequest(message.CmndData)
            channel <- request
        }

    }
}
