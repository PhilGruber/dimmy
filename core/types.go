package core

const CycleLength = 200 // ms

type DeviceConfig struct {
	Type    string         `yaml:"type"`
	Topic   string         `yaml:"topic"`
	Name    string         `yaml:"name"`
	Label   string         `yaml:"label"`
	Emoji   string         `yaml:"emoji"`
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
	Hidden            *bool              `yaml:"hidden"`
	Timeout           *int               `yaml:"timeout"`
	TargetOnDuration  *int               `yaml:"targetOnDuration"`
	TargetOffDuration *int               `yaml:"targetOffDuration"`
	Target            *string            `yaml:"target"`
	Min               *int               `yaml:"min"`
	Max               *int               `yaml:"max"`
	Devices           *[]string          `yaml:"devices,flow"`
	Margin            *float64           `yaml:"margin"`
	Transition        *bool              `yaml:"transition"`
	Commands          *map[string]string `yaml:"commands,flow"`
	Fields            *[]string          `yaml:"fields,flow"`
	History           *bool              `yaml:"history"`
}

type ServerConfig struct {
	Port       int            `yaml:"port"`
	MqttServer string         `yaml:"mqtt_server"`
	WebRoot    string         `yaml:"webroot"`
	Devices    []DeviceConfig `yaml:"devices"`
	Rules      []RuleConfig   `yaml:"rules"`
	Panels     []PanelConfig  `yaml:"panels"`
}

func ToPtr[T any](v T) *T {
	return &v
}
