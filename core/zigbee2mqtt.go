package core

type Zigbee2MqttMessageUpdate struct {
	state string
}

type Zigbee2MqttMessage struct {
	Battery         int                      `json:"battery"`
	Linkquality     int                      `json:"linkquality"`
	UpdateAvailable bool                     `json:""`
	Update          Zigbee2MqttMessageUpdate `json:"update"`
}

type Zigbee2MqttLightMessage struct {
	State      string `json:"state"`
	Brightness int    `json:"brightness"`
	Transition *int   `json:"transition"`
}
