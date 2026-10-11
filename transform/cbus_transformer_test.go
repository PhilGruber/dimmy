package transform

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCbusEncode(t *testing.T) {
	for _, tc := range []struct {
		name string
		data map[string]any
		want string
	}{
		{"on", map[string]any{"state": "ON"}, `{"command":"on"}`},
		{"off", map[string]any{"state": "OFF"}, `{"command":"off"}`},
		{"boolean", map[string]any{"state": true}, `{"command":"on"}`},
		{"form value", map[string]any{"state": "false"}, `{"command":"off"}`},
		{"full brightness", map[string]any{"brightness": 255, "transition": 4}, `{"command":"ramp","value":255,"time":4}`},
		{"light value", map[string]any{"value": 128, "transition": 2}, `{"command":"ramp","value":128,"time":2}`},
		{"immediate", map[string]any{"brightness": 127}, `{"command":"ramp","value":127,"time":0}`},
		{"fade off", map[string]any{"brightness": 0, "transition": 4}, `{"command":"ramp","value":0,"time":4}`},
		{"off with retained brightness", map[string]any{"state": "OFF", "brightness": 255, "transition": 4}, `{"command":"ramp","value":0,"time":4}`},
		{"JSON numbers", map[string]any{"brightness": float64(255), "transition": float64(4)}, `{"command":"ramp","value":255,"time":4}`},
		{"generic strings", map[string]any{"value": "255", "time": "4"}, `{"command":"ramp","value":255,"time":4}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			topic, payload, err := (CbusTransformer{}).Encode("cbus/lighting/1", tc.data)
			require.NoError(t, err)
			require.Equal(t, "cbus/lighting/1/set", topic)
			require.JSONEq(t, tc.want, string(payload))
		})
	}
}

func TestCbusRejectsInvalidCommands(t *testing.T) {
	for _, data := range []map[string]any{
		{}, {"state": "TOGGLE"}, {"brightness": -1}, {"brightness": 256},
		{"brightness": 1.5}, {"brightness": nil}, {"brightness": "bad"},
		{"brightness": 255, "transition": -1}, {"brightness": 255, "transition": "bad"},
	} {
		_, _, err := (CbusTransformer{}).Encode("cbus/lighting/1", data)
		require.Error(t, err, "data: %v", data)
	}
}

func TestCbusNoPolling(t *testing.T) {
	topic, payload, err := (CbusTransformer{}).PollMessage("cbus/lighting/1", map[string]any{"brightness": 0})
	require.NoError(t, err)
	require.Empty(t, topic)
	require.Empty(t, payload)
}

func TestCbusDecode(t *testing.T) {
	for _, tc := range []struct {
		payload string
		level   int
		state   string
	}{
		{`{"command":"on"}`, 255, "ON"},
		{`{"command":"off"}`, 0, "OFF"},
		{`{"command":"ramp","value":127,"time":4}`, 127, "ON"},
		{`{"command":"ramp","value":0,"time":4}`, 0, "OFF"},
	} {
		got, err := (CbusTransformer{}).Decode([]byte(tc.payload))
		require.NoError(t, err)
		require.Equal(t, tc.level, got["brightness"])
		require.Equal(t, tc.level, got["value"])
		require.Equal(t, tc.state, got["state"])
		if got["command"] == "ramp" {
			require.Equal(t, 4, got["transition"])
		}
	}
	for _, payload := range []string{"invalid", "null", `{}`, `{"command":"toggle"}`, `{"command":"ramp"}`, `{"command":"ramp","value":256}`, `{"command":"ramp","value":0,"time":-1}`} {
		_, err := (CbusTransformer{}).Decode([]byte(payload))
		require.Error(t, err)
	}
}
