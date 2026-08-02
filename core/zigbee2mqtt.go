package core

type Zigbee2MqttMessageUpdate struct {
	State string `json:"state,omitempty"`
}

type Zigbee2MqttMessage struct {
	Battery         *int                     `json:"battery,omitempty"`
	LinkQuality     *int                     `json:"linkquality,omitempty"`
	UpdateAvailable *bool                    `json:"-"`
	Update          Zigbee2MqttMessageUpdate `json:"update,omitempty"`
}

type Zigbee2MqttLightMessage struct {
	Zigbee2MqttMessage
	State      *string `json:"state,omitempty"`
	Brightness *int    `json:"brightness,omitempty"`
	Transition *int    `json:"transition,omitempty"`
}

type Zigbee2MqttBlindMessage struct {
	Position *int `json:"position"`
}

type Zigbee2MqttBlindStatusMessage struct {
	Zigbee2MqttMessage
	Zigbee2MqttBlindMessage
	State *string `json:"state"`
}
