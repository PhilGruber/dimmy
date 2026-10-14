package devices

import (
	"github.com/PhilGruber/dimmy/core"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestOptionalDeviceConfigDoesNotPanic(t *testing.T) {
	require.NotPanics(t, func() { NewIrControl(core.DeviceConfig{Name: "ir"}) })
	require.NotPanics(t, func() { NewShell(core.DeviceConfig{Name: "shell", Options: &core.ConfigOptions{}}) })
}

func TestInvalidRequestsDoNotPanic(t *testing.T) {
	device := NewDevice(core.DeviceConfig{Name: "generic"})
	require.NotPanics(t, func() { device.ProcessRequest(core.SwitchRequest{}) })
	ir := NewIrControl(core.DeviceConfig{Name: "ir"})
	require.NotPanics(t, func() { ir.SetReceiverValue("command", 42) })
	light := NewLight(core.DeviceConfig{Name: "light"})
	require.NotPanics(t, func() { light.SetReceiverValue("brightness", "invalid") })
	require.NotPanics(t, func() { light.SetReceiverValue("duration", nil) })
}

func TestConditionUnsupportedValuesDoNotPanic(t *testing.T) {
	for _, tc := range []struct {
		value, target any
		operator      string
	}{
		{1, int64(2), ">"},
		{[]any{1}, []any{1}, "=="},
		{map[string]any{"x": 1}, map[string]any{"x": 1}, "!="},
	} {
		c := condition{LastValue: tc.value, Value: tc.target, Operator: tc.operator}
		require.NotPanics(t, func() { require.False(t, c.check()) })
	}
}
