package core

import "os"

const CycleLength = 200 // ms

type DeviceConfig struct {
	Type    string         `yaml:"type"`
	Topic   string         `yaml:"topic"`
	Name    string         `yaml:"name"`
	Label   string         `yaml:"label"`
	Icon    string         `yaml:"icon"`
	Options *configOptions `yaml:"options"`
}

type RuleConfig struct {
	Triggers  []TriggerConfig  `yaml:"triggers"`
	Receivers []ReceiverConfig `yaml:"receivers"`
}

type TriggerConfig struct {
	DeviceName string                  `yaml:"device"`
	Key        string                  `yaml:"key"`
	Active     bool                    `yaml:"active"`
	Condition  ReceiverConditionConfig `yaml:"condition"`
}

type ReceiverConditionConfig struct {
	Operator string `yaml:"operator"`
	Value    any    `yaml:"value"`
	Delay    *int   `yaml:"delay"`
}

type ReceiverConfig struct {
	DeviceName string `yaml:"device"`
	Key        string `yaml:"key"`
	Value      string `yaml:"value"`
}

type PanelConfig struct {
	Label   string   `yaml:"label"`
	Devices []string `yaml:"devices"`
}

type configOptions struct {
	Hidden     *bool              `yaml:"hidden"`
	Transition *bool              `yaml:"transition"`
	Commands   *map[string]string `yaml:"commands,flow"`
	Sensors    *[]Sensor          `yaml:"sensors,flow"`
	Controls   *[]Control         `yaml:"controls,flow"`
	Devices    *[]string          `yaml:"devices,flow"`

	History *bool `yaml:"history"`

	/* deprecated */
	Fields *[]string `yaml:"fields,flow"`
	/* deprecated */
	Timeout *int `yaml:"timeout"`
	/* deprecated */
	Target *string `yaml:"target"`
	/* deprecated */
	Min *int `yaml:"min"`
	/* deprecated */
	Max *int `yaml:"max"`
	/* deprecated */
	Margin *float64 `yaml:"margin"`
}

type ControlType string

const (
	ControlTypeScale    ControlType = "scale"
	ControlTypeBool     ControlType = "bool"
	ControlTypeList     ControlType = "list"
	ControlTypeDimmable ControlType = "dimmable"
	ControlTypeColour   ControlType = "colour"
)

type Control struct {
	Name         string      `yaml:"name"`
	Icon         string      `yaml:"icon"`
	Type         ControlType `yaml:"type"`
	Hidden       bool        `yaml:"hidden"`
	Min          *int        `yaml:"min"`
	Max          *int        `yaml:"max"`
	NeedsSending bool
	Value        any
}

type Sensor struct {
	Name      string   `yaml:"name"`
	Icon      string   `yaml:"icon"`
	Values    []string `yaml:"values"`
	Hidden    bool     `yaml:"hidden"`
	ShowSince *string  `yaml:"show_since"`
	History   *bool    `yaml:"history"`
}

func (s *Sensor) GetIconHtml() string {
	if _, err := os.Stat("html/assets/icons/" + s.Icon); err == nil {
		return "<img class='icon' src='" + s.Icon + "' alt='" + s.Name + "'>"
	}
	return s.Icon
}

type ServerConfig struct {
	Port       int            `yaml:"port"`
	MqttServer string         `yaml:"mqtt_server"`
	WebRoot    string         `yaml:"webroot"`
	Lat        float64        `yaml:"latitude"`
	Lon        float64        `yaml:"longitude"`
	Devices    []DeviceConfig `yaml:"devices"`
	Rules      []RuleConfig   `yaml:"rules"`
	Panels     []PanelConfig  `yaml:"panels"`
}

func ToPtr[T any](v T) *T {
	return &v
}
