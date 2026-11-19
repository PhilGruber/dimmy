package devices

import (
	"fmt"
	"testing"

	"github.com/PhilGruber/dimmy/core"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func controlRule(device DeviceInterface, key string, value any) *Rule {
	return NewRule(core.RuleConfig{Triggers: []core.TriggerConfig{{
		DeviceName: device.GetName(), Key: key,
		Condition: core.ReceiverConditionConfig{Operator: "==", Value: value},
	}}}, map[string]DeviceInterface{device.GetName(): device})
}

func TestCbusACRulesOnOffOn(t *testing.T) {
	var rules []core.RuleConfig
	require.NoError(t, yaml.Unmarshal([]byte(`
- triggers:
    - device: ac
      key: brightness
      condition: {operator: "==", value: "0"}
  receivers: [{device: Wohnzimmer-AC, key: command, value: ❌}]
- triggers:
    - device: ac
      key: brightness
      condition: {operator: "==", value: "100"}
    - device: Wohnzimmer-Temperatur
      key: temperature
      condition: {operator: ">=", value: "21.5"}
  receivers: [{device: Wohnzimmer-AC, key: command, value: ❄️}]
- triggers:
    - device: ac
      key: brightness
      condition: {operator: "==", value: "100"}
    - device: Wohnzimmer-Temperatur
      key: temperature
      condition: {operator: "<", value: "21.5"}
  receivers: [{device: Wohnzimmer-AC, key: command, value: 🔥}]
`), &rules))
	light := NewLight(core.DeviceConfig{Name: "ac", Topic: "cbus/livingroom/ac", Protocol: "cbus", Options: &core.ConfigOptions{Hidden: core.ToPtr(true)}})
	sensors := []core.Sensor{{Name: "temperature"}}
	temperature := NewDevice(core.DeviceConfig{Name: "Wohnzimmer-Temperatur", Options: &core.ConfigOptions{Sensors: &sensors}})
	commands := core.Commands{{Name: "❌", Value: "OFF"}, {Name: "❄️", Value: "COOL"}, {Name: "🔥", Value: "HEAT"}}
	ir := NewIrControl(core.DeviceConfig{Name: "Wohnzimmer-AC", Options: &core.ConfigOptions{Commands: &commands}})
	devices := map[string]DeviceInterface{light.Name: light, temperature.Name: temperature, ir.Name: ir}
	activeRules := make([]*Rule, 0)
	for _, config := range rules {
		activeRules = append(activeRules, NewRule(config, devices))
	}
	temperature.setSensorValue("temperature", 22.0)
	channel := make(chan core.SwitchRequest, 10)
	for _, step := range []struct{ command, expected string }{{"on", "❄️"}, {"off", "❌"}, {"on", "❄️"}} {
		light.GetMessageHandler(nil, light)(nil, &mockMessage{payload: []byte(fmt.Sprintf(`{"command":%q,"source_unit":19,"type":"lighting"}`, step.command))})
		fired := 0
		for _, rule := range activeRules {
			if rule.CheckTriggers() {
				rule.Fire(channel)
				rule.ClearTriggers()
				fired++
			}
		}
		require.Equal(t, 1, fired, "command %s", step.command)
		request := <-channel
		ir.ProcessRequest(request)
		require.Equal(t, step.expected, ir.LastIcon)
		for _, rule := range activeRules {
			require.False(t, rule.CheckTriggers(), "must not repeat without a new reading")
		}
	}
	temperature.setSensorValue("temperature", 20.0)
	require.True(t, activeRules[2].CheckTriggers(), "temperature crossing threshold selects heating while AC remains on")
	temperature.setSensorValue("temperature", 21.5)
	require.True(t, activeRules[1].CheckTriggers(), "threshold belongs to cooling rule")
	require.False(t, activeRules[2].CheckTriggers())
}

func TestTemperatureRulesUseExistingReading(t *testing.T) {
	sensors := []core.Sensor{{Name: "temperature"}}
	d := NewDevice(core.DeviceConfig{Name: "temperature", Options: &core.ConfigOptions{Sensors: &sensors}})
	d.setSensorValue("temperature", 22.0)
	timestamp := d.Values["temperature"].LastChanged
	rule := controlRule(d, "temperature", "22")
	require.True(t, rule.CheckTriggers())
	require.Equal(t, timestamp, *rule.Triggers[0].Condition.LastChanged)
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

func TestCbusBrightnessTriggers(t *testing.T) {
	for _, kind := range []string{"light", "zlight"} {
		for _, transition := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/transition=%t", kind, transition), func(t *testing.T) {
				config := core.DeviceConfig{Name: "Cbus light", Protocol: "cbus", Options: &core.ConfigOptions{Transition: &transition}}
				var light DeviceInterface = NewLight(config)
				if kind == "zlight" {
					light = NewZLight(config)
				}
				require.Contains(t, light.GetTriggers(), "brightness")
				rule := NewRule(core.RuleConfig{Triggers: []core.TriggerConfig{{DeviceName: config.Name, Key: "brightness", Condition: core.ReceiverConditionConfig{Operator: ">", Value: 0}}}}, map[string]DeviceInterface{config.Name: light})
				handler := light.GetMessageHandler(nil, light)
				handler(nil, &mockMessage{payload: []byte(`{"command":"ramp","value":128,"time":0}`)})
				require.True(t, rule.CheckTriggers())
				handler(nil, &mockMessage{payload: []byte(`{"command":"off"}`)})
				require.False(t, rule.CheckTriggers())
			})
		}
	}
}
