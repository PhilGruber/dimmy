package transform

import (
	"fmt"
	"math"
	"strconv"
)

type Protocol interface {
	Encode(string, map[string]any) (string, []byte, error)
	Decode([]byte) (map[string]any, error)
	PollMessage(string, map[string]any) (string, []byte, error)
}

// IntValue accepts integer values from typed messages and JSON maps.
func IntValue(value any) (int, bool) {
	switch value := value.(type) {
	case int:
		return value, true
	case float64:
		if math.Trunc(value) != value {
			return 0, false
		}
		result, err := strconv.Atoi(strconv.FormatFloat(value, 'f', -1, 64))
		if err != nil {
			return 0, false
		}
		return result, true
	default:
		return 0, false
	}
}

// UnsupportedProtocol reports configuration errors without panicking in MQTT callbacks.
type UnsupportedProtocol struct{ Name string }

func (t UnsupportedProtocol) Encode(string, map[string]any) (string, []byte, error) {
	return "", nil, fmt.Errorf("unsupported protocol %q", t.Name)
}
func (t UnsupportedProtocol) Decode([]byte) (map[string]any, error) {
	return nil, fmt.Errorf("unsupported protocol %q", t.Name)
}
func (t UnsupportedProtocol) PollMessage(string, map[string]any) (string, []byte, error) {
	return "", nil, fmt.Errorf("unsupported protocol %q", t.Name)
}
