package devices

import (
	"encoding/json"
	core "github.com/PhilGruber/dimmy/core"
	"log"
	"math"
	"regexp"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Light struct {
	Dimmable
}

type lightStateMessage struct {
	Value int    `json:"Dimmer"`
	State string `json:"POWER"`
}

func makeLight(config core.DeviceConfig) Light {
	d := Light{}
	d.MqttTopic = config.Topic

	var re = regexp.MustCompile("^cmnd/(.+)/dimmer$")
	d.MqttState = re.ReplaceAllString(d.MqttTopic, "tele/$1/STATE")

	d.Current = 0
	d.Target = 0

	d.Hidden = false
	d.Min = 0
	d.Max = 100

	if config.Options != nil {
		if config.Options.Hidden != nil {
			d.Hidden = *config.Options.Hidden
		}
		if config.Options.Min != nil {
			d.Min = *config.Options.Min
		}
		if config.Options.Max != nil {
			d.Max = *config.Options.Max
		}
	}

	tt := time.Now()
	d.LastChanged = &tt
	d.Type = "light"
	return d
}

func NewLight(config core.DeviceConfig) *Light {
	d := makeLight(config)
	return &d
}

func (l *Light) PublishValue(mqtt mqtt.Client) {
	tt := time.Now()
	newVal := int(math.Round(l.Current))
	if newVal != l.LastSent {
		l.LastChanged = &tt
		l.LastSent = newVal
		mqtt.Publish(l.MqttTopic, 0, false, strconv.Itoa(int(math.Round(l.Current))))
	}
}

func (l *Light) GetMessageHandler(channel chan core.SwitchRequest, light DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := mqttMessage.Payload()
		value, err := strconv.Atoi(string(payload))
		state := value > 0
		if err != nil {
			var data lightStateMessage
			err := json.Unmarshal(payload, &data)
			if err != nil {
				log.Println("Error: " + err.Error())
				return
			}
			value = data.Value
			state = data.State == "ON"
		}
		if !state {
			value = 0
		}
		log.Printf("Received state value %d from %s\n", value, light.GetMqttStateTopic())
		if l.GetTarget() == math.Round(l.GetCurrent()) {
			l.setTarget(float64(value))
		}
		l.setCurrent(float64(value))

	}
}
