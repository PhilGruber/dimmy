package devices

import (
	"encoding/json"
	"fmt"
	"github.com/PhilGruber/dimmy/core"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"log"
)

type ZTemperature struct {
	Temperature
}

func MakeZTemperature(config core.DeviceConfig) ZTemperature {
	t := ZTemperature{}
	t.Emoji = "🌡️"
	t.setBaseConfig(config)

	t.Current = 0
	t.Humidity = 0
	t.Type = "temperature"

	return t
}

func NewZTemperature(config core.DeviceConfig) *ZTemperature {
	t := MakeZTemperature(config)
	return &t
}

type ZTemperatureMessage struct {
	core.Zigbee2MqttMessage
	Temperature float64  `json:"temperature"`
	Humidity    *float64 `json:"humidity"`
}

func (t *ZTemperature) GetMessageHandler(channel chan core.SwitchRequest, temperature DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := mqttMessage.Payload()
		var data ZTemperatureMessage
		err := json.Unmarshal(payload, &data)
		if err != nil {
			log.Printf("[%32s] Received invalid json payload: %v\n\tError: %s ", t.GetName(), t.MqttTopic, payload, err.Error())
			return
		}
		str := fmt.Sprintf("[%32s] Received new temperature %.2f", t.Name, data.Temperature)
		if data.Humidity != nil {
			str += fmt.Sprintf(" Humidity: %.2f", *data.Humidity)
		}
		log.Println(str)
		if data.Temperature != 0 {
			t.SetCurrent(data.Temperature)
		}
		if data.Humidity != nil {
			t.Humidity = *data.Humidity
		}
		t.addDataLog(data.Temperature, data.Humidity)

		t.setBatteryLevel(data.Battery)
		t.setLinkQuality(data.LinkQuality)
	}
}

func (t *ZTemperature) GetHumidity() float64 {
	return t.Humidity
}
