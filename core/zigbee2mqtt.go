package core

type Zigbee2MqttMessageUpdate struct {
	State string `json:"state"`
}

type Zigbee2MqttMessage struct {
	Battery         *int                     `json:"battery"`
	LinkQuality     *int                     `json:"linkquality"`
	UpdateAvailable *bool                    `json:""`
	Update          Zigbee2MqttMessageUpdate `json:"update"`
}

type Zigbee2MqttLightMessage struct {
	Zigbee2MqttMessage
	State      *string `json:"state"`
	Brightness *int    `json:"brightness"`
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
