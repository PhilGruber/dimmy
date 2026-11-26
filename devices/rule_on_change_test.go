package devices

import (
	"testing"

	"github.com/PhilGruber/dimmy/core"
	"github.com/stretchr/testify/require"
)

func TestRuleOnChangeUsesOnlyFlaggedTrigger(t *testing.T) {
	light := NewLight(core.DeviceConfig{Name: "light"})
	sensors := []core.Sensor{{Name: "temperature"}}
	temperature := NewDevice(core.DeviceConfig{Name: "temperature", Options: &core.ConfigOptions{Sensors: &sensors}})
	rule := NewRule(core.RuleConfig{Triggers: []core.TriggerConfig{
		{DeviceName: "light", Key: "brightness", OnChange: true, Condition: core.ReceiverConditionConfig{Operator: "==", Value: "100"}},
		{DeviceName: "temperature", Key: "temperature", Condition: core.ReceiverConditionConfig{Operator: ">=", Value: "21.5"}},
	}}, map[string]DeviceInterface{"light": light, "temperature": temperature})
	require.NotNil(t, rule)
	require.False(t, rule.CheckTriggers())
	light.SetCurrent(100)
	temperature.setSensorValue("temperature", 20.0)
	require.False(t, rule.CheckTriggers())
	temperature.setSensorValue("temperature", 22.0)
	require.True(t, rule.CheckTriggers(), "pending change can fire when the other condition matches")
	rule.Fire(make(chan core.SwitchRequest, 1))
	rule.ClearTriggers()
	require.False(t, rule.CheckTriggers())
	light.SetCurrent(100)
	temperature.setSensorValue("temperature", 23.0)
	require.False(t, rule.CheckTriggers(), "identical flagged value and changing other triggers cannot retrigger")
	light.SetCurrent(0)
	require.False(t, rule.CheckTriggers())
	light.SetCurrent(100)
	require.True(t, rule.CheckTriggers(), "changing away and back permits another firing")
}

func TestRuleOnChangeRemembersClearedEventValue(t *testing.T) {
	sensors := []core.Sensor{{Name: "action"}}
	d := NewDevice(core.DeviceConfig{Name: "button", Options: &core.ConfigOptions{Sensors: &sensors}})
	rule := NewRule(core.RuleConfig{Triggers: []core.TriggerConfig{{DeviceName: "button", Key: "action", OnChange: true, Condition: core.ReceiverConditionConfig{Operator: "==", Value: "single"}}}}, map[string]DeviceInterface{"button": d})
	d.setSensorValue("action", "single")
	require.True(t, rule.CheckTriggers())
	rule.Fire(make(chan core.SwitchRequest, 1))
	rule.ClearTriggers()
	d.setSensorValue("action", "single")
	require.False(t, rule.CheckTriggers())
	d.setSensorValue("action", "double")
	require.False(t, rule.CheckTriggers())
	d.setSensorValue("action", "single")
	require.True(t, rule.CheckTriggers())
}

func TestRuleRejectsMultipleOnChangeTriggers(t *testing.T) {
	d := NewLight(core.DeviceConfig{Name: "light"})
	trigger := core.TriggerConfig{DeviceName: "light", Key: "brightness", OnChange: true}
	require.Nil(t, NewRule(core.RuleConfig{Triggers: []core.TriggerConfig{trigger, trigger}}, map[string]DeviceInterface{"light": d}))
	require.Empty(t, d.rules)
}
