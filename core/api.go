package core

type SwitchRequest struct {
	Device   string  `json:"device"`
	Value    float64 `json:"value"`
	Duration int     `json:"duration"`
}
