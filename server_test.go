package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

	form := url.Values{"topic": {topic}, "name": {"Kitchen Sensor"}, "type": {"generic-device"}}
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
	require.Equal(t, "zigbee", config.Devices[0].Transformer)
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
	light := dimmyDevices.NewLight(core.DeviceConfig{Name: name, Topic: "cbus/JERV/backyard/ceiling", Transformer: "cbus"})
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
