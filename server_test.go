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
	dimmyDevices "github.com/PhilGruber/dimmy/devices"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

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

	form := url.Values{"topic": {topic}, "name": {"Kitchen Sensor"}}
	request := httptest.NewRequest(http.MethodPost, "/devices/save", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	server.SaveUnknownDevice().ServeHTTP(response, request)

	require.Equal(t, http.StatusCreated, response.Code)
	require.NotContains(t, server.unknownDevices, topic)
	require.Same(t, device, server.devices["Kitchen Sensor"])
	require.Equal(t, "Kitchen Sensor", device.GetName())

	data, err := os.ReadFile(filename)
	require.NoError(t, err)
	var config core.ServerConfig
	require.NoError(t, yaml.Unmarshal(data, &config))
	require.Len(t, config.Devices, 1)
	require.Equal(t, "Kitchen Sensor", config.Devices[0].Name)
	require.Equal(t, "device", config.Devices[0].Type)
	require.Equal(t, topic, config.Devices[0].Topic)
	require.Len(t, *config.Devices[0].Options.Sensors, 2)
}
