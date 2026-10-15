package transform

import (
	"math"
	"testing"
)

func TestIntValue(t *testing.T) {
	for _, tc := range []struct {
		value any
		want  int
		ok    bool
	}{
		{int(127), 127, true}, {float64(127), 127, true}, {float64(0), 0, true},
		{127.5, 0, false}, {"127", 0, false}, {nil, 0, false},
		{math.Inf(1), 0, false}, {math.NaN(), 0, false}, {1e100, 0, false},
	} {
		got, ok := IntValue(tc.value)
		if got != tc.want || ok != tc.ok {
			t.Errorf("IntValue(%v) = %d, %v; want %d, %v", tc.value, got, ok, tc.want, tc.ok)
		}
	}
}

func TestTasmotaDecode(t *testing.T) {
	for _, tc := range []struct {
		payload string
		value   int
		state   string
	}{
		{"50", 50, "ON"}, {"0", 0, "OFF"},
		{`{"Dimmer":50,"POWER":"ON"}`, 50, "ON"},
		{`{"Dimmer":50,"POWER":"OFF"}`, 50, "OFF"},
	} {
		data, err := (TasmotaProtocol{}).Decode([]byte(tc.payload))
		if err != nil {
			t.Fatal(err)
		}
		if data["brightness"] != tc.value || data["state"] != tc.state {
			t.Errorf("Decode(%s) = %v", tc.payload, data)
		}
	}
}

func TestTasmotaInvalidEncodeDoesNotPanic(t *testing.T) {
	for _, value := range []any{nil, "invalid", true, 1.5} {
		_, _, err := (TasmotaProtocol{}).Encode("cmnd/light", map[string]any{"brightness": value})
		if err == nil {
			t.Fatalf("expected error for value %v", value)
		}
	}
}

func TestProtocolStateTopics(t *testing.T) {
	for _, tc := range []struct {
		protocol    Protocol
		topic, want string
	}{
		{TasmotaProtocol{}, "cmnd/kitchen", "tele/kitchen/STATE"},
		{TasmotaProtocol{}, "tele/kitchen/STATE", "tele/kitchen/STATE"},
		{ZigbeeProtocol{}, "cmnd/kitchen", "cmnd/kitchen"},
		{CbusProtocol{}, "cbus/lighting/1", "cbus/lighting/1"},
	} {
		if got := tc.protocol.StateTopic(tc.topic); got != tc.want {
			t.Fatalf("StateTopic(%q) = %q; want %q", tc.topic, got, tc.want)
		}
	}
}

func TestTasmotaBrightnessWireFormat(t *testing.T) {
	topic, payload, err := (TasmotaProtocol{}).Encode("cmnd/kitchen", map[string]any{"brightness": 50})
	if err != nil || topic != "cmnd/kitchen/dimmer" || string(payload) != "50" {
		t.Fatalf("unexpected command: %s %s %v", topic, payload, err)
	}
}
