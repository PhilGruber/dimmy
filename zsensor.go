package main

import (
    "encoding/json"
    mqtt "github.com/eclipse/paho.mqtt.golang"
    "log"
    "math"
    "strconv"
)

type ZSensor struct {
    Dimmable

    TargetDevice string
    TargetOnDuration int
    TargetOffDuration int
    Timeout int
}
func MakeZSensor(config map[string]string) ZSensor {
    s := ZSensor{}
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

    s.Hidden = false
    if val, ok := config["hidden"]; ok {
        s.Hidden = val == "true"
    }

    s.Current = 0
    s.Target = 0
    s.Type = "sensor"
    return s
}

func NewZSensor(config map[string]string) *ZSensor {
    s := MakeZSensor(config)
    return &s
}

type ZSensorMessage struct {
    Zigbee2MqttMessage

    Occupancy bool`json:"occupancy"`
}

func (s *ZSensor) getTimeoutRequest() (SwitchRequest, bool) {
    var request SwitchRequest
    return request, false
}

func (s *ZSensor) getMessageHandler(channel chan SwitchRequest, sensor DeviceInterface) mqtt.MessageHandler {
    log.Println("Subscribing to " + sensor.getMqttTopic())
    return func (client mqtt.Client, mqttMessage mqtt.Message) {

        payload := mqttMessage.Payload()

        log.Printf("%s", payload)

        if sensor.getCurrent() == 0 {
            return
        }

        var data ZSensorMessage
        err := json.Unmarshal(payload, &data)
        if err != nil {
            log.Println("Error: " + err.Error())
            return
        }

        val := "on"
        if data.Occupancy {
            val = "on"
        } else {
        	val = "off"
        }
        log.Println(sensor.getMqttTopic() + " is " + val)
        request, ok := sensor.generateRequest(val)

        if ok {
            channel <- request
        }

    }
}

func (s *ZSensor) generateRequest(cmd string) (SwitchRequest, bool) {
    var request SwitchRequest
    request.Device   = s.TargetDevice
    if cmd == "on" {
        request.Value = int(math.Round(s.Current))
        request.Duration = s.TargetOnDuration
    } else {
        request.Value = 0
        request.Duration = s.TargetOffDuration
    }
    return request, true
}
