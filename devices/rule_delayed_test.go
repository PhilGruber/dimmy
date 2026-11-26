package devices

import (
	"testing"

	"github.com/PhilGruber/dimmy/core"
	"github.com/stretchr/testify/require"
)

func TestRuleFireKeepsMultipleImmediateControls(t *testing.T) {
	d := NewDevice(core.DeviceConfig{Name: "device"})
	rule := &Rule{Receivers: []Receiver{
		{Device: d, Key: "first", Value: "on"},
		{Device: d, Key: "second", Value: "on"},
		{Device: d, Key: "first", Value: "off", Delay: 600},
	}}
	channel := make(chan core.SwitchRequest, 3)
	rule.Fire(channel)
	require.Len(t, channel, 2)
	keys := make([]string, 0)
	for len(channel) > 0 {
		request := <-channel
		keys = append(keys, request.Key)
		require.Equal(t, "on", request.Value)
	}
	require.ElementsMatch(t, []string{"first", "second"}, keys)
}
