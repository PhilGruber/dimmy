package transform

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type CbusTransformer struct{}

// Encode sends commands to the configured topic with the /set suffix.
// Brightness/value is a raw C-Bus level (0–255); transition/time is in seconds.
func (t CbusTransformer) Encode(topic string, data map[string]any) (string, []byte, error) {
	message := make(map[string]any)
	level, dimmable := data["brightness"]
	if !dimmable {
		level, dimmable = data["value"]
	}
	if dimmable {
		value, ok := cbusInt(level)
		if !ok || value < 0 || value > 255 {
			return "", nil, fmt.Errorf("invalid C-Bus brightness %v: expected an integer from 0 to 255", level)
		}
		seconds := 0
		duration, present := data["transition"]
		if !present {
			duration, present = data["time"]
		}
		if present {
			seconds, ok = cbusInt(duration)
			if !ok || seconds < 0 {
				return "", nil, fmt.Errorf("invalid C-Bus ramp time %v: expected nonnegative seconds", duration)
			}
		}
		// OFF takes precedence over a retained brightness, as with Zigbee lights.
		if state, present := data["state"]; present {
			on, err := cbusState(state)
			if err != nil {
				return "", nil, err
			}
			if !on {
				value = 0
			}
		}
		message["command"] = "ramp"
		message["value"] = value
		message["time"] = seconds
	} else {
		on, err := cbusState(data["state"])
		if err != nil {
			return "", nil, err
		}
		command := "off"
		if on {
			command = "on"
		}
		message["command"] = command
	}
	payload, err := json.Marshal(message)
	return topic + "/set", payload, err
}

func cbusInt(value any) (int, bool) {
	if text, ok := value.(string); ok {
		result, err := strconv.Atoi(text)
		return result, err == nil
	}
	return IntValue(value)
}

func cbusState(value any) (bool, error) {
	if on, ok := value.(bool); ok {
		return on, nil
	}
	if state, ok := value.(string); ok {
		switch strings.ToLower(state) {
		case "on", "true":
			return true, nil
		case "off", "false":
			return false, nil
		}
	}
	return false, fmt.Errorf("invalid C-Bus state %v: expected ON or OFF", value)
}

// Decode normalizes C-Bus messages for light and generic-device handlers.
func (t CbusTransformer) Decode(data []byte) (map[string]any, error) {
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	value := 0
	switch result["command"] {
	case "on":
		value = 255
	case "off":
	case "ramp":
		var ok bool
		value, ok = cbusInt(result["value"])
		if !ok || value < 0 || value > 255 {
			return nil, fmt.Errorf("invalid C-Bus brightness %v: expected an integer from 0 to 255", result["value"])
		}
		if duration, present := result["time"]; present {
			seconds, ok := cbusInt(duration)
			if !ok || seconds < 0 {
				return nil, fmt.Errorf("invalid C-Bus ramp time %v: expected nonnegative seconds", duration)
			}
			result["transition"] = seconds
		}
	default:
		return nil, fmt.Errorf("unsupported C-Bus command %v", result["command"])
	}
	result["value"] = value
	result["brightness"] = value
	result["state"] = "OFF"
	if value > 0 {
		result["state"] = "ON"
	}
	return result, nil
}

func (t CbusTransformer) PollMessage(string, map[string]any) (string, []byte, error) {
	return "", nil, nil
}
