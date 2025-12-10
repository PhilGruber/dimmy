package core

type SwitchRequest struct {
	Device string `json:"device"`
	// Deprecated: key=command; value=value should be used
	Command  string `json:"command"`
	Key      string `json:"key"`
	Value    string `json:"value"`
	Duration int    `json:"duration"`
}
