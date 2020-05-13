package main

import (
    "strconv"
    "log"

    mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Thermostat struct {
    Device

    TargetDevice string
    TargetTemperature float64
    Margin float64
}

func MakeThermostat(config map[string]string) Thermostat {
    t := Thermostat{}
    t.MqttTopic = config["topic"]
    t.TargetDevice = config["target"]

    var val string
    var ok bool

    if val, ok = config["margin"]; !ok {
        val = "0.5"
    }
    t.Margin, _ = strconv.ParseFloat(val, 64)
    t.TargetTemperature = 18
    t.Current = 0
    t.Type = "thermostat"
    return t
}

func (t Thermostat) getMin() int {
    return 0
}

func (t Thermostat) getMax() int {
    return 99
}

func NewThermostat(config map[string]string) *Thermostat {
    s := MakeThermostat(config)
    return &s
}

func (t *Thermostat) PublishValue(mqtt mqtt.Client) {
}

func (t *Thermostat) processRequest(request SwitchRequest) {
    t.TargetTemperature = float64(request.Value)
}


func (t *Thermostat) generateRequest(payload string) (SwitchRequest, bool) {

    t.Current, _ = strconv.ParseFloat(payload, 64)

    var request SwitchRequest
    if t.Current < (t.TargetTemperature - t.Margin/2) {
        log.Printf("Too cold, turning on %s. Currently %.2f C, need %.2f C", t.TargetDevice, t.Current, t.TargetTemperature - t.Margin/2)
        request.Device   = t.TargetDevice
        request.Value    = 1
        request.Duration = 0

        return request, true

    } else if t.Current > (t.TargetTemperature + t.Margin/2) {
        log.Printf("Too hot, turning off %s. Currently %.2f C, need %.2f C", t.TargetDevice, t.Current, t.TargetTemperature + t.Margin/2)
        request.Device   = t.TargetDevice
        request.Value    = 0
        request.Duration = 0

        return request, true
    }

    return request, false
}

func (t Thermostat) UpdateValue() (float64, bool) {
    return 0, false
}

func ThermostatMessageHandler(channel chan SwitchRequest, thermostat DeviceInterface) mqtt.MessageHandler {
    return func (client mqtt.Client, mqttMessage mqtt.Message) {
        payload := string(mqttMessage.Payload())
        log.Println("Received new temperature: " + string(payload))

        if request, ok := thermostat.generateRequest(payload[:]); ok {
            channel <- request
        }
    }
}

