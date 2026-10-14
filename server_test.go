package main

import (
	"bytes"
	"encoding/json"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PhilGruber/dimmy/core"
	dimmyDevices "github.com/PhilGruber/dimmy/devices"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestInitializeRoomDashboards(t *testing.T) {
	config := &core.ServerConfig{
		Devices: []core.DeviceConfig{
			{Name: "kitchen-light", Type: "device", Room: "Kitchen"},
			{Name: "kitchen-sensor", Type: "device", Room: "Kitchen"},
			{Name: "bedroom-light", Type: "device", Room: "Bedroom"},
			{Name: "unassigned", Type: "device"},
			{Name: "reserved-default", Type: "device", Room: "default"},
			{Name: "reserved-all", Type: "device", Room: "all"},
		},
		Panels: []core.PanelConfig{{Label: "Configured", Devices: []string{"kitchen-light"}}},
	}
	s := &Server{}
	s.initialize(config)

	require.Len(t, s.dashboards, 4)
	require.NotContains(t, s.dashboards, "")
	for room, names := range map[string][]string{
		"Kitchen": {"kitchen-light", "kitchen-sensor"},
		"Bedroom": {"bedroom-light"},
	} {
		require.Len(t, s.dashboards[room], 1)
		panel := s.dashboards[room][0]
		require.Equal(t, room, panel.GetLabel())
		var actual []string
		for _, device := range panel.GetDevices() {
			actual = append(actual, device.GetName())
		}
		require.ElementsMatch(t, names, actual)
	}
	for _, name := range []string{"default", "all"} {
		require.Len(t, s.dashboards[name], 6)
		require.Equal(t, "Configured", s.dashboards[name][0].GetLabel())
		for _, panel := range s.dashboards[name] {
			require.NotContains(t, []string{"Kitchen", "Bedroom"}, panel.GetLabel())
		}
	}
}

func TestSaveUnknownDevice(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "dimmyd.conf.yaml")
	require.NoError(t, os.WriteFile(filename, []byte("mqtt_server: localhost\ndevices: []\n"), 0o600))

	topic := "zigbee/kitchen-sensor"
	device := dimmyDevices.NewDeviceFromMessage(topic, map[string]any{
		"temperature": 21.5,
		"humidity":    48.0,
	})
	server := &Server{
		devices:        make(map[string]dimmyDevices.DeviceInterface),
		unknownDevices: map[string]dimmyDevices.DeviceInterface{topic: device},
		config:         &core.ServerConfig{Filename: filename},
	}

	form := url.Values{"topic": {topic}, "name": {"Kitchen Sensor"}, "type": {"generic-device"}, "room": {" Kitchen "}, "protocol": {"cbus"}}
	request := httptest.NewRequest(http.MethodPost, "/devices/save", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	server.SaveUnknownDevice().ServeHTTP(response, request)

	require.Equal(t, http.StatusCreated, response.Code)
	require.NotContains(t, server.unknownDevices, topic)
	require.NotNil(t, server.devices["Kitchen Sensor"])
	require.Equal(t, "Kitchen Sensor", server.devices["Kitchen Sensor"].GetName())

	data, err := os.ReadFile(filename)
	require.NoError(t, err)
	var config core.ServerConfig
	require.NoError(t, yaml.Unmarshal(data, &config))
	require.Len(t, config.Devices, 1)
	require.Equal(t, "Kitchen Sensor", config.Devices[0].Name)
	require.Equal(t, "device", config.Devices[0].Type)
	require.Equal(t, topic, config.Devices[0].Topic)
	require.Equal(t, "cbus", config.Devices[0].Protocol)
	require.Equal(t, "Kitchen", config.Devices[0].Room)
	require.Equal(t, "Kitchen", server.devices["Kitchen Sensor"].GetRoom())
	require.Len(t, *config.Devices[0].Options.Sensors, 2)
}

func TestSaveUnknownDeviceInvalidTypeReleasesLock(t *testing.T) {
	for _, typ := range []string{"", "unsupported"} {
		t.Run(typ, func(t *testing.T) {
			topic := "zigbee/test"
			s := &Server{devices: make(map[string]dimmyDevices.DeviceInterface), unknownDevices: map[string]dimmyDevices.DeviceInterface{topic: dimmyDevices.NewDeviceFromMessage(topic, map[string]any{"temperature": 21.0})}}
			form := url.Values{"topic": {topic}, "name": {"test"}, "type": {typ}}
			req := httptest.NewRequest(http.MethodPost, "/devices/new-devices/save", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			res := httptest.NewRecorder()
			require.NotPanics(t, func() { s.SaveUnknownDevice().ServeHTTP(res, req) })
			require.Equal(t, http.StatusUnprocessableEntity, res.Code)
			require.True(t, s.mutex.TryLock(), "request left the server locked")
			s.mutex.Unlock()
			require.Empty(t, s.devices)
			require.Contains(t, s.unknownDevices, topic)
		})
	}
}

func TestReceiveRequestLimitsAndLogging(t *testing.T) {
	var logs bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(previous)
	for _, tc := range []struct {
		name, body string
		status     int
		queued     bool
	}{
		{"valid", `{"device":"light","value":"private-value"}`, http.StatusOK, true},
		{"invalid", `private-value`, http.StatusBadRequest, false},
		{"oversized", strings.Repeat("x", maxRequestBodyBytes+1), http.StatusRequestEntityTooLarge, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &Server{channel: make(chan core.SwitchRequest, 1)}
			req := httptest.NewRequest(http.MethodPost, "/api/switch", strings.NewReader(tc.body))
			req.ContentLength = -1 // Also protect chunked/unknown-length requests.
			res := httptest.NewRecorder()
			s.ReceiveRequest().ServeHTTP(res, req)
			require.Equal(t, tc.status, res.Code)
			require.Equal(t, tc.queued, len(s.channel) == 1)
		})
	}
	require.NotContains(t, logs.String(), "private-value")
	require.Empty(t, logs.String())
}

func TestHTTPServerLimitsBodies(t *testing.T) {
	var readErr error
	var size int
	srv := newHTTPServer(":0", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		size, readErr = len(data), err
	}))
	srv.Handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", maxRequestBodyBytes+1))))
	var tooLarge *http.MaxBytesError
	require.ErrorAs(t, readErr, &tooLarge)
	require.Equal(t, maxRequestBodyBytes, size)
	require.Positive(t, srv.ReadHeaderTimeout)
	require.Positive(t, srv.ReadTimeout)
	require.Positive(t, srv.WriteTimeout)
	require.Positive(t, srv.IdleTimeout)
}

func TestDashboardCbusLightValue(t *testing.T) {
	name := "CBUS Light"
	light := dimmyDevices.NewLight(core.DeviceConfig{Name: name, Topic: "cbus/JERV/backyard/ceiling", Protocol: "cbus"})
	light.SetCurrent(50)
	server := &Server{
		devices:    map[string]dimmyDevices.DeviceInterface{name: light},
		dashboards: map[string][]dimmyDevices.Panel{"default": {dimmyDevices.NewPanelFromDevice(light)}},
	}
	response := httptest.NewRecorder()
	server.ShowDashboard("html")(response, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), "id='value_"+name+"'>50%</span>")
	require.NotContains(t, response.Body.String(), "parseInt(")
	status := httptest.NewRecorder()
	server.ShowStatus(&server.devices)(status, httptest.NewRequest(http.MethodGet, "/api/status", nil))
	var state map[string]struct {
		Value float64 `json:"value"`
	}
	require.NoError(t, json.Unmarshal(status.Body.Bytes(), &state))
	require.Equal(t, float64(50), state[name].Value)
}

func TestUnknownDevicesRoomOptions(t *testing.T) {
	s := &Server{
		devices: map[string]dimmyDevices.DeviceInterface{
			"one":   dimmyDevices.NewDevice(core.DeviceConfig{Name: "one", Room: "Kitchen"}),
			"two":   dimmyDevices.NewDevice(core.DeviceConfig{Name: "two", Room: "Kitchen"}),
			"three": dimmyDevices.NewDevice(core.DeviceConfig{Name: "three", Room: "Bedroom"}),
		},
	}
	res := httptest.NewRecorder()
	s.ShowUnknownDevices("html").ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/devices/new-devices", nil))
	require.Equal(t, http.StatusOK, res.Code)
	body := res.Body.String()
	require.Equal(t, 1, strings.Count(body, `<option value="Kitchen">`))
	require.Contains(t, body, `<option value="Bedroom">`)
	require.Less(t, strings.Index(body, `<option value="Bedroom">`), strings.Index(body, `<option value="Kitchen">`))
}

func TestClearMultipleFiredRules(t *testing.T) {
	first := &dimmyDevices.Rule{SingleUse: true}
	retained := &dimmyDevices.Rule{}
	last := &dimmyDevices.Rule{SingleUse: true}
	s := &Server{rules: []*dimmyDevices.Rule{first, retained, last}}
	require.NotPanics(t, func() { s.clearFiredRules([]int{0, 1, 2}) })
	require.Equal(t, []*dimmyDevices.Rule{retained}, s.rules)
}

func TestSaveRulesWithInvalidExistingControls(t *testing.T) {
	controls := []core.Control{{Name: "power"}}
	sensors := []core.Sensor{{Name: "temperature"}}
	device := dimmyDevices.NewDevice(core.DeviceConfig{Name: "sensor", Options: &core.ConfigOptions{Controls: &controls, Sensors: &sensors}})
	valid := core.RuleConfig{
		Triggers:  []core.TriggerConfig{{DeviceName: "sensor", Key: "temperature", Active: true, Condition: core.ReceiverConditionConfig{Operator: "==", Value: "20"}}},
		Receivers: []core.ReceiverConfig{{DeviceName: "sensor", Key: "power", Value: "on"}},
	}
	broken := valid
	broken.Receivers = []core.ReceiverConfig{{DeviceName: "sensor", Key: "removed", Value: "on"}}
	broken.Triggers = append([]core.TriggerConfig(nil), valid.Triggers...)
	broken.Triggers[0].Condition.Value = 20
	otherBroken := valid
	otherBroken.Receivers = []core.ReceiverConfig{{DeviceName: "missing-device", Key: "power", Value: "on"}}
	for _, tc := range []struct {
		name    string
		updated []core.RuleConfig
		status  int
	}{
		{"delete broken rule while another remains", []core.RuleConfig{otherBroken}, http.StatusOK},
		{"repair broken rule while another remains", []core.RuleConfig{valid, otherBroken}, http.StatusOK},
		{"delete all rules", []core.RuleConfig{}, http.StatusOK},
		{"reject newly introduced broken rule", []core.RuleConfig{broken, otherBroken, broken}, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "rules.yaml")
			original := []core.RuleConfig{broken, otherBroken}
			require.NoError(t, core.SaveRulesConfig(filename, original))
			s := &Server{devices: map[string]dimmyDevices.DeviceInterface{"sensor": device}, config: &core.ServerConfig{Rules: original, RulesFilename: filename}}
			body, err := json.Marshal(tc.updated)
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodPut, "/api/rules", bytes.NewReader(body))
			res := httptest.NewRecorder()
			s.SaveRules().ServeHTTP(res, req)
			require.Equal(t, tc.status, res.Code, res.Body.String())
			if tc.status == http.StatusOK {
				var persisted []core.RuleConfig
				data, err := os.ReadFile(filename)
				require.NoError(t, err)
				require.NoError(t, yaml.Unmarshal(data, &persisted))
				require.Equal(t, tc.updated, persisted)
			} else {
				require.Equal(t, original, s.config.Rules)
			}
		})
	}
}

func TestEditRulesShowsConfiguredMissingDevices(t *testing.T) {
	s := &Server{devices: map[string]dimmyDevices.DeviceInterface{}, config: &core.ServerConfig{Rules: []core.RuleConfig{
		{Receivers: []core.ReceiverConfig{{DeviceName: "missing-lamp", Key: "old-control", Value: "on"}}},
		{Receivers: []core.ReceiverConfig{{DeviceName: "second-missing-lamp", Key: "other-control", Value: "on"}}},
	}}}
	res := httptest.NewRecorder()
	s.EditRules("html").ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/rules", nil))
	require.Equal(t, http.StatusOK, res.Code)
	require.Contains(t, res.Body.String(), "missing-lamp (unavailable)")
	require.Contains(t, res.Body.String(), "old-control: on")
	require.Contains(t, res.Body.String(), `data-edit-rule="0"`)
	require.Contains(t, res.Body.String(), `data-delete-rule="1"`)
}

// Embedded interfaces make unexpected MQTT operations fail the test.
type subscriptionClient struct {
	mqtt.Client
	handlers      map[string]mqtt.MessageHandler
	subscriptions []string
}

func (c *subscriptionClient) Subscribe(topic string, qos byte, handler mqtt.MessageHandler) mqtt.Token {
	c.handlers[topic] = handler
	c.subscriptions = append(c.subscriptions, topic)
	return subscriptionToken{}
}

type subscriptionToken struct{}

func (subscriptionToken) Wait() bool                     { return true }
func (subscriptionToken) WaitTimeout(time.Duration) bool { return true }
func (subscriptionToken) Error() error                   { return nil }
func (subscriptionToken) Done() <-chan struct{} {
	done := make(chan struct{})
	close(done)
	return done
}

type subscribedDevice struct {
	dimmyDevices.DeviceInterface
	messages int
	polls    int
}

func (d *subscribedDevice) GetMessageHandler(chan core.SwitchRequest, dimmyDevices.DeviceInterface) mqtt.MessageHandler {
	return func(mqtt.Client, mqtt.Message) { d.messages++ }
}
func (d *subscribedDevice) PollValue(mqtt.Client) { d.polls++ }

func TestMqttReconnectRestoresSubscriptions(t *testing.T) {
	first := &subscribedDevice{DeviceInterface: dimmyDevices.NewLight(core.DeviceConfig{Name: "first", Topic: "cmnd/shared"})}
	second := &subscribedDevice{DeviceInterface: dimmyDevices.NewLight(core.DeviceConfig{Name: "second", Topic: "cmnd/shared"})}
	members := []string{"first"}
	all := map[string]dimmyDevices.DeviceInterface{"first": first, "second": second}
	all["group"] = dimmyDevices.NewGroup(core.DeviceConfig{Name: "group", Options: &core.ConfigOptions{Devices: &members}}, all)
	s := &Server{devices: all, config: &core.ServerConfig{}}
	opts := s.mqttOptions("localhost", "test")
	require.NotNil(t, opts.OnConnect)
	require.NotNil(t, opts.OnConnectionLost)
	require.NotNil(t, opts.OnReconnecting)
	c := &subscriptionClient{handlers: make(map[string]mqtt.MessageHandler)}
	opts.OnConnect(c)
	require.ElementsMatch(t, []string{"#", "tele/shared/STATE"}, c.subscriptions)
	c.handlers["tele/shared/STATE"](c, nil)
	require.Equal(t, 1, first.messages)
	require.Equal(t, 1, second.messages)
	// Simulate a clean broker session after reconnect, including a saved device.
	third := &subscribedDevice{DeviceInterface: dimmyDevices.NewLight(core.DeviceConfig{Name: "third", Topic: "cmnd/new"})}
	s.devices["third"] = third
	c.handlers = make(map[string]mqtt.MessageHandler)
	c.subscriptions = nil
	opts.OnConnect(c)
	require.ElementsMatch(t, []string{"#", "tele/shared/STATE", "tele/new/STATE"}, c.subscriptions)
	c.handlers["tele/shared/STATE"](c, nil)
	c.handlers["tele/new/STATE"](c, nil)
	require.Equal(t, 2, first.messages)
	require.Equal(t, 2, second.messages)
	require.Equal(t, 1, third.messages)
	require.Equal(t, 2, first.polls)
	require.Equal(t, 1, third.polls)
}
