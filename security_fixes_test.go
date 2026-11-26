package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PhilGruber/dimmy/core"
	dimmyDevices "github.com/PhilGruber/dimmy/devices"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/require"
)

func TestDashboardEscapesDeviceIcons(t *testing.T) {
	payload := `<img src=x onerror="alert(1)">`
	device := dimmyDevices.NewLight(core.DeviceConfig{Name: "lamp", Icon: payload})
	s := &Server{dashboards: map[string][]dimmyDevices.Panel{"default": {dimmyDevices.NewPanelFromDevice(device)}}}
	res := httptest.NewRecorder()
	s.ShowDashboard("html")(res, httptest.NewRequest(http.MethodGet, "/", nil))
	require.NotContains(t, res.Body.String(), payload)
	require.Contains(t, res.Body.String(), "&lt;img")
	require.False(t, strings.Contains(res.Body.String(), `<img src=x`))
}

func TestCrossOriginStateChanges(t *testing.T) {
	for _, tc := range []struct {
		name, method, origin, site, referer string
		status                              int
	}{
		{"same origin", "POST", "http://dimmy.local", "same-origin", "", 204},
		{"CLI", "POST", "", "", "", 204},
		{"hostile form", "POST", "http://attacker.example", "cross-site", "", 403},
		{"hostile PUT", "PUT", "http://attacker.example", "", "", 403},
		{"null origin", "POST", "null", "", "", 403},
		{"fetch metadata", "POST", "", "cross-site", "", 403},
		{"same site is not same origin", "POST", "", "same-site", "", 403},
		{"forged port", "POST", "http://dimmy.local:9999", "", "", 403},
		{"legacy referer", "POST", "", "", "http://attacker.example/page", 403},
		{"GET", "GET", "http://attacker.example", "cross-site", "", 204},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			srv := newHTTPServer(":0", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(204) }))
			r := httptest.NewRequest(tc.method, "http://dimmy.local/devices/edit/save", nil)
			r.Header.Set("Origin", tc.origin)
			r.Header.Set("Sec-Fetch-Site", tc.site)
			r.Header.Set("Referer", tc.referer)
			res := httptest.NewRecorder()
			srv.Handler.ServeHTTP(res, r)
			require.Equal(t, tc.status, res.Code)
			require.Equal(t, tc.status == 204, called)
		})
	}
	s := &Server{channel: make(chan core.SwitchRequest, 1)}
	res := httptest.NewRecorder()
	s.ReceiveRequest().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/switch", nil))
	require.Equal(t, http.StatusMethodNotAllowed, res.Code)
	require.Empty(t, s.channel)
}

type discoveryMessage struct {
	mqtt.Message
	topic string
}

func (m discoveryMessage) Topic() string   { return m.topic }
func (m discoveryMessage) Payload() []byte { return []byte(`{"temperature":22}`) }

func TestUnknownDeviceExpiry(t *testing.T) {
	s := &Server{devices: make(map[string]dimmyDevices.DeviceInterface), unknownDevices: make(map[string]dimmyDevices.DeviceInterface)}
	handler := s.DetectDevice()
	handler(nil, discoveryMessage{topic: "zigbee/old"})
	handler(nil, discoveryMessage{topic: "zigbee/refreshed"})
	now := time.Now()
	s.unknownLastSeen["zigbee/old"] = now.Add(-3*time.Hour - time.Nanosecond)
	s.unknownLastSeen["zigbee/refreshed"] = now.Add(-2 * time.Hour)
	original := s.unknownDevices["zigbee/refreshed"]
	handler(nil, discoveryMessage{topic: "zigbee/refreshed"})
	require.Same(t, original, s.unknownDevices["zigbee/refreshed"])
	s.pruneUnknownDevices(now)
	require.NotContains(t, s.unknownDevices, "zigbee/old")
	require.NotContains(t, s.unknownLastSeen, "zigbee/old")
	require.Contains(t, s.unknownDevices, "zigbee/refreshed")
	s.unknownLastSeen["zigbee/refreshed"] = now.Add(-3 * time.Hour)
	s.pruneUnknownDevices(now)
	require.Contains(t, s.unknownDevices, "zigbee/refreshed", "exactly three hours is not expired")
	s.pruneUnknownDevices(now.Add(time.Nanosecond))
	require.Empty(t, s.unknownDevices)
	handler(nil, discoveryMessage{topic: "zigbee/refreshed"})
	require.Contains(t, s.unknownDevices, "zigbee/refreshed", "expired devices can be rediscovered")
}
