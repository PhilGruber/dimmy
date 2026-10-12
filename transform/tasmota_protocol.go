package transform

import (
	"encoding/json"
	"strconv"
)

type TasmotaProtocol struct {
	Protocol
}

func (t TasmotaProtocol) Encode(topic string, data map[string]any) (string, []byte, error) {
	result := strconv.Itoa(data["value"].(int))
	return topic + "/dimmer", []byte(result), nil
}

func (t TasmotaProtocol) Decode(data []byte) (map[string]any, error) {
	value, err := strconv.Atoi(string(data))
	result := make(map[string]any)
	result["value"] = value
	result["state"] = "OFF"
	if value > 0 {
		result["state"] = "ON"
	}
	if err != nil {
		var msg struct {
			Value int    `json:"Dimmer"`
			State string `json:"POWER"`
		}
		err = json.Unmarshal(data, &msg)
		if err != nil {
			return nil, err
		}
		result["value"] = msg.Value
		result["state"] = msg.State
	}
	return result, err
}

func (t TasmotaProtocol) PollMessage(topic string, data map[string]any) (string, []byte, error) {
	return "", []byte(""), nil
}
