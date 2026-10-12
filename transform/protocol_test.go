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
		if data["value"] != tc.value || data["state"] != tc.state {
			t.Errorf("Decode(%s) = %v", tc.payload, data)
		}
	}
}
