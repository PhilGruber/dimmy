package core

type SwitchRequest struct {
	Device   string `json:"device"`
	Command  string `json:"command"`
	Value    string `json:"value"`
	Duration int    `json:"duration"`
}
