package transform

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"math"
	"strconv"
	"time"
)

type Protocol interface {
	StateTopic(string) string
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

func (t UnsupportedProtocol) StateTopic(topic string) string { return topic }

// Poll sends the protocol-specific query, if the protocol supports polling.
func Poll(client mqtt.Client, protocol Protocol, topic string, fields map[string]any) error {
	topic, payload, err := protocol.PollMessage(topic, fields)
	if err != nil || topic == "" {
		return err
	}
	token := client.Publish(topic, 0, false, payload)
	if !token.WaitTimeout(10 * time.Second) {
		return fmt.Errorf("poll timed out: %s", topic)
	}
	return token.Error()
}
