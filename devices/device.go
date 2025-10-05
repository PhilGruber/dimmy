package devices

import (
	"encoding/json"
	"html/template"
	"log"
	"os"
	"sync"
	"time"

	"github.com/PhilGruber/dimmy/core"
	"github.com/google/uuid"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type DeviceInterface interface {
	UpdateValue() (float64, bool)
	GenerateRequest(string) (core.SwitchRequest, bool)

	GetMqttTopic() string
	GetMqttStateTopic() string
	GetType() string
	GetName() string
	GetLabel() string
	GetMax() int
	GetMin() int
	GetHidden() bool
	GetCurrent() float64
	SetCurrent(float64)
	GetEmoji() string
	ProcessRequest(core.SwitchRequest)
	GetMessageHandler(chan core.SwitchRequest, DeviceInterface) mqtt.MessageHandler
	GetStateMessageHandler(chan core.SwitchRequest, DeviceInterface) mqtt.MessageHandler
	GetTriggers() []string
	GetReceivers() []string
	SetReceiverValue(string, any)
	ClearTrigger(string)
	Lock()
	Unlock()
	AddRule(*Rule)
	RemoveRule(*Rule)
	IsPersistent(string) bool
	HasReceivers() bool
	GetIconHtml() template.HTML

	PublishValue(mqtt.Client)
	PollValue(mqtt.Client)
}

type Device struct {
	Name        string
	MqttTopic   string     `json:"-"`
	MqttState   string     `json:"-"`
	Current     float64    `json:"value"`
	LastChanged *time.Time `json:"lastUpdate"`
	Type        string
	Hidden      bool
	Label       string
	Icon        string
	Triggers    []string
	Receivers   []string
	mutex       *sync.RWMutex
	rules       []*Rule

	persistentFields []string

	LinkQuality *int `json:"linkquality"`
	Battery     *int `json:"battery"`
}

func (d *Device) setBaseConfig(config core.DeviceConfig) {
	d.MqttTopic = config.Topic
	d.Current = 0
	if config.Icon != "" {
		d.Icon = config.Icon
	}

	if config.Name != "" {
		d.Name = config.Name
	} else {
		d.Name = uuid.NewString()
	}

	if config.Label != "" {
		d.Label = config.Label
	} else {
		d.Label = d.Name
	}

	past := time.Unix(0, 0)
	d.LastChanged = &past

	d.Hidden = false
	if config.Options != nil {
		if config.Options.Hidden != nil {
			d.Hidden = *config.Options.Hidden
		}
	}

	d.persistentFields = []string{"battery"}

	d.mutex = new(sync.RWMutex)
}

func (d *Device) GetCurrent() float64 {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.Current
}

func (d *Device) SetCurrent(current float64) {
	now := time.Now()
	d.mutex.Lock()
	d.LastChanged = &now
	d.Current = current
	d.mutex.Unlock()
	d.UpdateRules("value", current)
}

func (d *Device) GetType() string {
	return d.Type
}

func (d *Device) GetMqttTopic() string {
	return d.MqttTopic
}

func (d *Device) GetMqttStateTopic() string {
	return d.MqttState
}

func (d *Device) GetName() string {
	return d.Name
}

func (d *Device) GetLabel() string {
	if d.Label != "" {
		return d.Label
	}
	return d.Name
}

func (d *Device) Lock() {
	d.mutex.RLock()
}

func (d *Device) Unlock() {
	d.mutex.RUnlock()
}

func (d *Device) GenerateRequest(cmd string) (core.SwitchRequest, bool) {
	var r core.SwitchRequest
	return r, false
}

func (d *Device) PublishValue(mqtt.Client) {
}

func (d *Device) PollValue(mqtt.Client) {
}

func (d *Device) GetMessageHandler(channel chan core.SwitchRequest, sensor DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		payload := mqttMessage.Payload()
		var data map[string]any
		err := json.Unmarshal(payload, &data)
		if err != nil {
			log.Printf("[%32s] Error: %d\n", d.Name, err.Error())
			return
		}

		d.parseDefaultValues(data)
	}
}

func (d *Device) GetStateMessageHandler(channel chan core.SwitchRequest, sensor DeviceInterface) mqtt.MessageHandler {
	return func(client mqtt.Client, mqttMessage mqtt.Message) {
		log.Printf("[%32s] Received state message from %s\n", d.GetName(), d.MqttState)
	}
}

func (d *Device) GetTriggers() []string {
	return d.Triggers
}

func (d *Device) GetReceivers() []string {
	return d.Receivers
}

func (d *Device) SetReceiverValue(key string, value any) {
}

func (d *Device) ClearTrigger(key string) {
}

func (d *Device) GetHidden() bool {
	return d.Hidden
}

func (d *Device) GetEmoji() string {
	return d.Icon
}

func (d *Device) GetMax() int {
	return 1
}
func (d *Device) GetMin() int {
	return 0
}
func (d *Device) ProcessRequest(request core.SwitchRequest) {
}

func (d *Device) setBatteryLevel(battery *int) {
	if battery == nil {
		return
	}
	d.Battery = battery
}

func (d *Device) setLinkQuality(linkQuality *int) {
	if linkQuality == nil {
		return
	}
	d.LinkQuality = linkQuality
}

func (d *Device) parseDefaultValues(data map[string]any) {
	if battery, ok := data["battery"]; ok {
		if batteryInt, ok := battery.(int); ok {
			d.setBatteryLevel(&batteryInt)
		}
	}
	if linkQuality, ok := data["linkquality"]; ok {
		if linkQualityInt, ok := linkQuality.(int); ok {
			d.setLinkQuality(&linkQualityInt)
		}
	}
}

func (d *Device) UpdateRule(rule *Rule, field string, value any) {
	now := time.Now()
	for t, trigger := range rule.Triggers {
		if trigger.Device.GetName() == d.GetName() && trigger.Key == field {
			rule.Triggers[t].Condition.LastValue = value
			rule.Triggers[t].Condition.LastChanged = &now
		}
	}
}

func (d *Device) UpdateRules(field string, value any) {
	for r := range d.rules {
		d.UpdateRule(d.rules[r], field, value)
	}
}

func (d *Device) AddRule(rule *Rule) {
	d.rules = append(d.rules, rule)
}

func (d *Device) RemoveRule(rule *Rule) {
	for i, r := range d.rules {
		if r == rule {
			d.rules = append(d.rules[:i], d.rules[i+1:]...)
		}
	}
}

func (d *Device) HasReceivers() bool {
	return len(d.Receivers) > 0
}

func (d *Device) IsPersistent(field string) bool {
	for _, f := range d.persistentFields {
		if f == field {
			return true
		}
	}
	return false
}

func (d *Device) GetIconHtml() template.HTML {
	// TODO: prevent escaping from path
	webroot := "/usr/share/dimmy"
	if _, err := os.Stat(webroot + "/assets/icons/" + d.Icon); err == nil {
		return template.HTML("<img class='icon' src='/assets/icons/" + d.Icon + "' alt='" + d.GetLabel() + "'>")
	}
	return template.HTML(d.Icon)
}
