package core

type SwitchRequest struct {
	Device   string  `json:"device"`
	Command  string  `json:"command"`
	Value    float64 `json:"value"`
	Duration int     `json:"duration"`
}
