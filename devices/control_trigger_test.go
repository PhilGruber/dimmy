package devices

import (
	"testing"

	"github.com/PhilGruber/dimmy/core"
	"github.com/stretchr/testify/require"
)

func controlRule(device DeviceInterface, key string, value any) *Rule {
	return NewRule(core.RuleConfig{Triggers: []core.TriggerConfig{{
		DeviceName: device.GetName(), Key: key,
		Condition: core.ReceiverConditionConfig{Operator: "==", Value: value},
	}}}, map[string]DeviceInterface{device.GetName(): device})
}

func TestGenericControlTriggers(t *testing.T) {
	for _, discovered := range []bool{false, true} {
		t.Run(map[bool]string{false: "configured", true: "discovered"}[discovered], func(t *testing.T) {
			var device *GenericDevice
			if discovered {
				device = NewDeviceFromMessage("zigbee/remote", map[string]any{"brightness": 10})
			} else {
				controls := []core.Control{{Name: "brightness"}}
				device = NewDevice(core.DeviceConfig{Name: "remote", Options: &core.ConfigOptions{Controls: &controls}})
			}
			require.Contains(t, device.GetTriggers(), "brightness")
			rule := controlRule(device, "brightness", "50")
			require.False(t, rule.CheckTriggers())
			device.GetMessageHandler(nil, device)(nil, &mockMessage{payload: []byte(`{"brightness":50}`)})
			require.True(t, rule.CheckTriggers(), "reported control value triggers rule")
			rule.ClearTriggers()
			require.False(t, rule.CheckTriggers())
			device.ProcessRequest(core.SwitchRequest{Key: "brightness", Value: "50"})
			require.True(t, rule.CheckTriggers(), "requested control value triggers rule")
			device.SetReceiverValue("brightness", 20)
			require.False(t, rule.CheckTriggers(), "nonmatching value stops matching")
		})
	}
}

func TestIRCommandTriggers(t *testing.T) {
	device := NewIrControl(irControlConfig(t, false))
	require.Contains(t, device.GetTriggers(), "command")
	rule := controlRule(device, "command", "power")
	device.ProcessRequest(core.SwitchRequest{Value: "power"})
	require.True(t, rule.CheckTriggers())
	rule.ClearTriggers()
	require.False(t, rule.CheckTriggers())
	device.SetReceiverValue("command", "power")
	require.True(t, rule.CheckTriggers())
	device.ProcessRequest(core.SwitchRequest{Value: "mute"})
	require.False(t, rule.CheckTriggers())
}
