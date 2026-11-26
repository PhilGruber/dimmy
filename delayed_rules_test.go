package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PhilGruber/dimmy/core"
	dimmyDevices "github.com/PhilGruber/dimmy/devices"
	"github.com/stretchr/testify/require"
)

func TestDelayedControlsUseSingleUseRule(t *testing.T) {
	source := dimmyDevices.NewLight(core.DeviceConfig{Name: "button"})
	lamp := dimmyDevices.NewLight(core.DeviceConfig{Name: "lamp"})
	other := dimmyDevices.NewLight(core.DeviceConfig{Name: "other"})
	clock := dimmyDevices.NewDimmyTime(core.DeviceConfig{Name: "time"}, 0, 0)
	s := &Server{devices: map[string]dimmyDevices.DeviceInterface{"button": source, "lamp": lamp, "other": other, "time": clock}, channel: make(chan core.SwitchRequest, 10)}
	rule := dimmyDevices.NewRule(core.RuleConfig{
		Triggers: []core.TriggerConfig{{DeviceName: "button", Key: "brightness", OnChange: true, Condition: core.ReceiverConditionConfig{Operator: "==", Value: "100"}}},
		Receivers: []core.ReceiverConfig{
			{DeviceName: "lamp", Key: "brightness", Value: "100"},
			{DeviceName: "lamp", Key: "brightness", Value: "0", Delay: 600},
			{DeviceName: "lamp", Key: "duration", Value: "2", Delay: 600},
			{DeviceName: "other", Key: "brightness", Value: "0", Delay: 600},
			{DeviceName: "other", Key: "brightness", Value: "100", Delay: 1200},
		},
	}, s.devices)
	s.rules = []*dimmyDevices.Rule{rule}
	source.SetCurrent(100)
	now := time.Now().Truncate(time.Second)
	s.processRules(now)
	require.Len(t, s.channel, 1)
	request := <-s.channel
	require.Equal(t, "lamp", request.Device)
	require.Equal(t, "100", request.Value)
	require.Len(t, s.rules, 3)
	require.True(t, s.rules[1].SingleUse)
	require.True(t, s.rules[2].SingleUse)
	require.Equal(t, 4, len(s.rules[1].Receivers)+len(s.rules[2].Receivers))
	s.processRules(now.Add(time.Second))
	require.Empty(t, s.channel)
	require.Len(t, s.rules, 3)
	// Advance the time-device trigger values to the scheduled deadline.
	for _, trigger := range clock.CreateTriggersFromTime(now.Add(10 * time.Minute)) {
		clock.UpdateRules(trigger.Key, trigger.Condition.Value)
	}
	s.processRules(now.Add(10 * time.Minute))
	require.Len(t, s.channel, 2)
	requests := make(map[string]core.SwitchRequest)
	for len(s.channel) > 0 {
		request := <-s.channel
		requests[request.Device] = request
	}
	require.Equal(t, "0", requests["lamp"].Value)
	require.Equal(t, 2, requests["lamp"].Duration)
	require.Equal(t, "0", requests["other"].Value)
	require.Len(t, s.rules, 2, "only the fired scheduled rule is removed")
	s.processRules(now.Add(10*time.Minute + time.Second))
	require.Empty(t, s.channel)
	for _, trigger := range clock.CreateTriggersFromTime(now.Add(20 * time.Minute)) {
		clock.UpdateRules(trigger.Key, trigger.Condition.Value)
	}
	s.processRules(now.Add(20 * time.Minute))
	require.Len(t, s.channel, 1)
	request = <-s.channel
	require.Equal(t, "other", request.Device)
	require.Equal(t, "100", request.Value)
	require.Len(t, s.rules, 1)
}

func TestSaveRulesKeepsScheduledControls(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "rules.yaml")
	require.NoError(t, os.WriteFile(filename, []byte("[]\n"), 0600))
	scheduled := &dimmyDevices.Rule{SingleUse: true}
	s := &Server{rules: []*dimmyDevices.Rule{scheduled}, config: &core.ServerConfig{RulesFilename: filename}}
	response := httptest.NewRecorder()
	s.SaveRules().ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/api/rules", strings.NewReader("[]")))
	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, []*dimmyDevices.Rule{scheduled}, s.rules)
}
