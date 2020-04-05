package main

type SwitchRequest struct {
    Device string `json:"device"`
    Command string `json:"command"`
    Value int `json:"value"`
    Duration int `json:"duration"`
}
