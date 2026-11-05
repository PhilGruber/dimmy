package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PhilGruber/dimmy/core"
	"github.com/stretchr/testify/require"
)

func TestDeviceEditor(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(filename, []byte("devices:\n  - name: lamp\n    type: light\n    topic: cmnd/lamp\n    room: Bedroom\n    options:\n      min: 10\n"), 0600))
	s := &Server{config: &core.ServerConfig{Filename: filename}}
	page := httptest.NewRecorder()
	s.EditDevices("html").ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/devices/edit", nil))
	require.Equal(t, http.StatusOK, page.Code)
	require.Contains(t, page.Body.String(), `id="save-device-form"`)
	require.Contains(t, page.Body.String(), `"Name":"lamp"`)
	require.Contains(t, page.Body.String(), `<option value="Bedroom">`)
	list := httptest.NewRecorder()
	s.EditDevices("html").ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/devices", nil))
	require.Equal(t, http.StatusOK, list.Code)
	for _, property := range []string{"Label", "MQTT topic", "Type", "Protocol", "Icon"} {
		require.Contains(t, list.Body.String(), "<th>"+property+"</th>")
	}
	require.NotContains(t, list.Body.String(), "<th>Hidden</th>")
	require.NotContains(t, list.Body.String(), "<th>Name</th>")
	require.NotContains(t, list.Body.String(), "<th>Room</th>")
	require.Contains(t, list.Body.String(), `href="/devices/edit?name=lamp"`)
	require.NotContains(t, list.Body.String(), `id="save-device-form"`)
	for _, tc := range []struct {
		name, typ, protocol string
		status              int
	}{
		{"missing", "light", "zigbee", http.StatusNotFound},
		{"lamp", "invalid", "zigbee", http.StatusBadRequest},
		{"lamp", "light", "invalid", http.StatusBadRequest},
		{"lamp", "light", "cbus", http.StatusOK},
	} {
		form := url.Values{"name": {tc.name}, "type": {tc.typ}, "protocol": {tc.protocol}, "room": {"Kitchen"}}
		req := httptest.NewRequest(http.MethodPost, "/devices/edit/save", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		res := httptest.NewRecorder()
		s.SaveDeviceEdits().ServeHTTP(res, req)
		require.Equal(t, tc.status, res.Code, res.Body.String())
	}
	devices, err := core.LoadConfiguredDevices(filename)
	require.NoError(t, err)
	require.Equal(t, "Kitchen", devices[0].Room)
	require.Equal(t, "cbus", devices[0].Protocol)
	require.Equal(t, 10, *devices[0].Options.Min)
	page = httptest.NewRecorder()
	s.EditDevices("html").ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/devices/edit", nil))
	require.Contains(t, page.Body.String(), `"Room":"Kitchen"`)
	for _, bounds := range []url.Values{
		{"min": {"bad"}, "max": {"200"}},
		{"min": {"200"}, "max": {"10"}},
		{"min": {"1.5"}, "max": {"100"}},
		{"min": {"0"}, "max": {"0"}},
		{"min": {""}, "max": {""}},
	} {
		bounds.Set("name", "lamp")
		bounds.Set("type", "light")
		bounds.Set("protocol", "cbus")
		bounds.Set("room", "Kitchen")
		bounds.Set("label", "Desk light")
		bounds.Set("icon", "💡")
		bounds.Set("hidden", "true")
		req := httptest.NewRequest(http.MethodPost, "/devices/edit/save", strings.NewReader(bounds.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		res := httptest.NewRecorder()
		s.SaveDeviceEdits().ServeHTTP(res, req)
		if bounds.Get("min") != "0" && bounds.Get("min") != "" {
			require.Equal(t, http.StatusBadRequest, res.Code)
		} else {
			require.Equal(t, http.StatusOK, res.Code, res.Body.String())
		}
	}
	devices, err = core.LoadConfiguredDevices(filename)
	require.NoError(t, err)
	require.Equal(t, "Desk light", devices[0].Label)
	require.Equal(t, "💡", devices[0].Icon)
	require.True(t, *devices[0].Options.Hidden)
	require.Nil(t, devices[0].Options.Min)
	require.Nil(t, devices[0].Options.Max)
}

func TestDeviceIconChoices(t *testing.T) {
	webroot := t.TempDir()
	dir := filepath.Join(webroot, "assets", "icons")
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "air conditioner.png"), []byte("image"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("text"), 0644))
	choices := deviceIconChoices(webroot)
	require.Contains(t, choices, deviceIconChoice{Value: "💡"})
	require.Contains(t, choices, deviceIconChoice{Value: "air conditioner.png", URL: "/assets/icons/air%20conditioner.png"})
	for _, choice := range choices {
		require.NotEqual(t, "notes.txt", choice.Value)
	}
}
