package devices

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/PhilGruber/dimmy/core"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Blind struct {
	Device
	Min      int
	Max      int
	Position int

	needsSending bool
}

func NewBlind(config core.DeviceConfig) *Blind {
	d := makeBlind(config)
	return &d
}

func makeBlind(config core.DeviceConfig) Blind {
	d := Blind{}
	d.Icon = "🪟"
	d.setBaseConfig(config)
	d.MqttState = config.Topic

	d.Min = 0
	d.Max = 100
	d.Type = "blind"
	d.Receivers = []string{"position"}

	if config.Options != nil {
		if config.Options.Min != nil {
			d.Min = *config.Options.Min
		}
		if config.Options.Max != nil {
			d.Max = *config.Options.Max
		}
	}

	d.Triggers = []string{"position"}
	d.persistentFields = []string{"position"}

	tt := time.Now()
	d.LastChanged = &tt
	//	d.init()
	return d
}

func (b *Blind) PublishValue(mqtt mqtt.Client) {
	tt := time.Now()

	b.LastChanged = &tt

	msg := core.Zigbee2MqttBlindMessage{
		Position: &b.Position,
	}

	s, _ := json.Marshal(msg)
	mqtt.Publish(b.MqttTopic+"/set", 0, false, s)

	b.needsSending = false

}

func (b *Blind) PollValue(mqtt mqtt.Client) {
	msg := core.Zigbee2MqttBlindStatusMessage{}
	s, _ := json.Marshal(msg)
	log.Printf("[%32s] Polling %s\n", b.GetName(), b.MqttState)
	t := mqtt.Publish(b.MqttState+"/get", 0, false, s)
	if t.Wait() && t.Error() != nil {
		log.Println(t.Error())
	}
}

func (b *Blind) GetMessageHandler(channel chan core.SwitchRequest, sw DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := mqttMessage.Payload()
		var data core.Zigbee2MqttBlindStatusMessage
		err := json.Unmarshal(payload, &data)
		if err != nil {
			log.Println("Error: " + err.Error())
			return
		}
		if data.Position == nil {
			return
		}
		fmt.Printf("[%32s] Received state Position %d\n", b.GetName(), *data.Position)
		b.Position = *data.Position
		if data.State != nil {
			if *data.State == "CLOSE" {
				b.SetCurrent(0)
			}
			if *data.State == "OPEN" {
				b.SetCurrent(100)
			}
		}
		if data.Battery != nil {
			b.setBatteryLevel(data.Battery)
		}
		if data.LinkQuality != nil {
			b.setLinkQuality(data.LinkQuality)
		}
	}
}

func (b *Blind) ProcessRequest(request core.SwitchRequest) {
	relative := false
	if request.Value[0] == '+' || request.Value[0] == '-' {
		relative = true
	}
	val, _ := strconv.ParseFloat(request.Value, 64)
	if relative {
		val = float64(b.Position) + val
	}
	val = math.Max(float64(b.Min), val)
	val = math.Min(float64(b.Max), val)
	fmt.Printf("Changing blind %s from %d to %d\n", b.Name, b.Position, int(val))
	b.Position = int(val)
	b.needsSending = true
}

func (b *Blind) UpdateValue() (float64, bool) {
	if b.needsSending {
		return float64(b.Position), true
	}
	return 0, false
}
