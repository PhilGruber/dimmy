package transform

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type TasmotaProtocol struct {
	Protocol
}

func (t TasmotaProtocol) Encode(topic string, data map[string]any) (string, []byte, error) {
	raw, present := data["brightness"]
	if !present {
		raw = data["value"]
	} // Generic controls can still be named value.
	value, ok := IntValue(raw)
	if !ok {
		return "", nil, fmt.Errorf("invalid Tasmota value: %v", raw)
	}
	result := strconv.Itoa(value)
	return topic + "/dimmer", []byte(result), nil
}

func (t TasmotaProtocol) Decode(data []byte) (map[string]any, error) {
	value, err := strconv.Atoi(string(data))
	result := make(map[string]any)
	result["brightness"] = value
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
		result["brightness"] = msg.Value
		result["state"] = msg.State
	}
	result["value"] = result["brightness"] // Preserve generic-device value controls.
	return result, err
}

func (t TasmotaProtocol) PollMessage(topic string, data map[string]any) (string, []byte, error) {
	return "", []byte(""), nil
}

func (t TasmotaProtocol) StateTopic(topic string) string {
	if strings.HasPrefix(topic, "cmnd/") && len(topic) > len("cmnd/") {
		return "tele/" + strings.TrimPrefix(topic, "cmnd/") + "/STATE"
	}
	return topic
}
