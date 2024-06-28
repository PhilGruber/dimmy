package devices

import (
	"encoding/json"
	"github.com/PhilGruber/dimmy/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
)

func MakeZTemperature(config core.DeviceConfig) ZTemperature {
	t := ZTemperature{}
	t.Name = config.Name
	t.MqttTopic = config.Topic

	t.Current = 0
	t.Humidity = 0
	t.Type = "temperature"
	return t
}

func NewZTemperature(config core.DeviceConfig) *ZTemperature {
	t := MakeZTemperature(config)
	return &t
}

type ZTemperature struct {
	Temperature
	Humidity float64
}

type ZTemperatureMessage struct {
	core.Zigbee2MqttMessage
	Temperature float64 `json:"temperature"`
	Humidity    float64 `json:"humidity"`
}

func (t *ZTemperature) GetMessageHandler(channel chan core.SwitchRequest, temperature DeviceInterface) mqtt.MessageHandler {
	log.Printf("Subscribing to temperature sensor on %s\n", t.MqttTopic)
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := mqttMessage.Payload()
		var data ZTemperatureMessage
		err := json.Unmarshal(payload, &data)
		if err != nil {
			log.Printf("Received invalid json payload from %s: %v\n\tError: %s ", t.MqttTopic, payload, err.Error())
			return
		}
		log.Printf("Received new temperature: %.2f Humidity: %.2f", data.Temperature, data.Humidity)
		t.setCurrent(data.Temperature)
		t.Humidity = data.Humidity
	}
}
