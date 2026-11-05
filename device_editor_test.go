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
	for _, property := range []string{"Name", "Label", "MQTT topic", "Type", "Room", "Protocol", "Icon"} {
		require.Contains(t, list.Body.String(), "<th>"+property+"</th>")
	}
	require.NotContains(t, list.Body.String(), "<th>Hidden</th>")
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
}
