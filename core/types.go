package core

import "os"

const CycleLength = 200 // ms

type DeviceConfig struct {
	Type    string         `yaml:"type"`
	Topic   string         `yaml:"topic"`
	Name    string         `yaml:"name"`
	Label   string         `yaml:"label,omitempty"`
	Icon    string         `yaml:"icon,omitempty"`
	Options *ConfigOptions `yaml:"options,omitempty"`
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

type ConfigOptions struct {
	Hidden           *bool              `yaml:"hidden,omitempty"`
	Transition       *bool              `yaml:"transition,omitempty"`
	Commands         *map[string]string `yaml:"commands,omitempty,flow"`
	Sensors          *[]Sensor          `yaml:"sensors,omitempty,flow"`
	Controls         *[]Control         `yaml:"controls,omitempty,flow"`
	Devices          *[]string          `yaml:"devices,omitempty,flow"`
	PreventResending bool               `yaml:"prevent_resending,omitempty"`

	History *bool `yaml:"history,omitempty"`

	/* deprecated */
	Fields *[]string `yaml:"fields,omitempty,flow"`
	/* deprecated */
	Timeout *int `yaml:"timeout,omitempty"`
	/* deprecated */
	Target *string `yaml:"target,omitempty"`
	/* deprecated */
	Min *int `yaml:"min,omitempty"`
	/* deprecated */
	Max *int `yaml:"max,omitempty"`
	/* deprecated */
	Margin *float64 `yaml:"margin,omitempty"`
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
	Icon         string      `yaml:"icon,omitempty"`
	Type         ControlType `yaml:"type"`
	Hidden       bool        `yaml:"hidden,omitempty"`
	Min          *int        `yaml:"min,omitempty"`
	Max          *int        `yaml:"max,omitempty"`
	NeedsSending bool        `yaml:"-"`
	Value        any         `yaml:"-"`
	Values       any         `yaml:"values,omitempty"`
}

func (c *Control) GetType() string {
	return string(c.Type)
}

type Sensor struct {
	Name       string            `yaml:"name"`
	Icon       string            `yaml:"icon,omitempty"`
	Values     []string          `yaml:"values,omitempty"`
	Hidden     bool              `yaml:"hidden,omitempty"`
	ShowSince  *string           `yaml:"show_since,omitempty"`
	History    *bool             `yaml:"history,omitempty"`
	ValueIcons map[string]string `yaml:"value_icons,omitempty"`
}

func GetIconHtml(icon string, name string) string {
	if len(icon) == 0 {
		return name
	}
	if _, err := os.Stat("html/assets/icons/" + icon); err == nil {
		return "<img class='icon' src='" + icon + "' alt='" + name + "'>"
	}
	return icon
}

func (c *Control) GetIconHtml() string {
	return GetIconHtml(c.Icon, c.Name)
}

func (s *Sensor) GetIconHtml() string {
	return GetIconHtml(s.Icon, s.Name)
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
	Filename   string         `yaml:"-"`
}

func ToPtr[T any](v T) *T {
	return &v
}
