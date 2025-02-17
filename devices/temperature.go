package devices

import (
	"github.com/PhilGruber/dimmy/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
	"strconv"
	"time"
)

type DataLog struct {
	Time        time.Time
	Temperature float64
	Humidity    *float64
}

type Temperature struct {
	Device
	Humidity float64
	DataLog  []DataLog `json:"history"`
}

func MakeTemperature(config core.DeviceConfig) Temperature {
	t := Temperature{}
	t.Emoji = "🌡️"
	t.setBaseConfig(config)

	t.Current = 0
	t.Type = "temperature"

	t.Triggers = []string{"temperature", "humidity"}

	return t
}

func NewTemperature(config core.DeviceConfig) *Temperature {
	s := MakeTemperature(config)
	return &s
}

func (t *Temperature) GetMin() int {
	return 0
}

func (t *Temperature) GetMax() int {
	return 99
}

func (t *Temperature) PublishValue(mqtt.Client) {
}

func (t *Temperature) ProcessRequest(core.SwitchRequest) {
}

func (t *Temperature) UpdateValue() (float64, bool) {
	return 0, false
}

func (t *Temperature) GetMqttStateTopic() string {
	return t.MqttTopic
}

func (t *Temperature) GetMessageHandler(chan core.SwitchRequest, DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := string(mqttMessage.Payload())
		log.Printf("Received new temperature for %s: %s", t.Name, payload)
		temperature, err := strconv.ParseFloat(payload[:], 64)
		if err != nil {
			log.Println("Received invalid temperature " + payload[:] + ": " + err.Error())
			return
		}
		t.SetCurrent(temperature)
		t.addDataLog(temperature, nil)
	}
}

func (t *Temperature) addDataLog(temperature float64, humidity *float64) {
	if len(t.DataLog) > 0 && t.DataLog[len(t.DataLog)-1].Time.Sub(time.Now()) < 1*time.Minute {
		if temperature != 0 {
			t.DataLog[len(t.DataLog)-1].Temperature = temperature
		}
		if humidity != nil {
			t.DataLog[len(t.DataLog)-1].Humidity = humidity
		}
		return
	}
	t.DataLog = append(t.DataLog, DataLog{Time: time.Now(), Temperature: temperature, Humidity: humidity})
}

func (t *Temperature) GetHumidity() float64 {
	return -1
}

func (t *Temperature) GetTriggerValue(trigger string) interface{} {
	if trigger == "temperature" {
		return t.GetCurrent()
	}
	if trigger == "humidity" {
		return t.GetHumidity()
	}
	return nil
}
