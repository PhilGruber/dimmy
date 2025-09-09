package core

type SwitchRequest struct {
	Device   string `json:"device"`
	Command  string `json:"command"`
	Key      string `json:"key"`
	Value    string `json:"value"`
	Duration int    `json:"duration"`
}
