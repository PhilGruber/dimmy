package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestUpdateDeviceFieldsPreservesConfiguration(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "devices.yaml")
	original := `# Server settings
mqtt_server: broker
custom_setting: keep
devices:
  - name: remote
    type: ircontrol
    topic: zigbee/remote
    room: Bedroom
    label: Air conditioner
    icon: ac.png
    custom_device_setting: keep
    options:
      commands:
        cold: IR_CODE
      prevent_resending: true
  - name: lamp
    type: light
    topic: cmnd/lamp
`
	require.NoError(t, os.WriteFile(filename, []byte(original), 0600))
	require.NoError(t, UpdateDeviceFields(filename, "remote", "ircontrol", "Kitchen", "zigbee"))
	data, err := os.ReadFile(filename)
	require.NoError(t, err)
	var config map[string]any
	require.NoError(t, yaml.Unmarshal(data, &config))
	require.Equal(t, "broker", config["mqtt_server"])
	require.Equal(t, "keep", config["custom_setting"])
	devices := config["devices"].([]any)
	remote := devices[0].(map[string]any)
	require.Equal(t, "Kitchen", remote["room"])
	require.Equal(t, "zigbee", remote["protocol"])
	require.Equal(t, "Air conditioner", remote["label"])
	require.Equal(t, "ac.png", remote["icon"])
	require.Equal(t, "keep", remote["custom_device_setting"])
	require.Equal(t, "zigbee/remote", remote["topic"])
	require.Equal(t, map[string]any{"commands": map[string]any{"cold": "IR_CODE"}, "prevent_resending": true}, remote["options"])
	require.Equal(t, map[string]any{"name": "lamp", "type": "light", "topic": "cmnd/lamp"}, devices[1])
	require.Contains(t, string(data), "# Server settings")
	info, err := os.Stat(filename)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	require.ErrorIs(t, UpdateDeviceFields(filename, "missing", "light", "", ""), ErrDeviceNotFound)
	after, err := os.ReadFile(filename)
	require.NoError(t, err)
	require.Equal(t, data, after)
	fields := DeviceEditorFields{Label: "Cooling", Icon: "❄️", Min: ToPtr(0), Max: ToPtr(200), Hidden: ToPtr(true)}
	require.NoError(t, UpdateDeviceFields(filename, "remote", "ircontrol", "Kitchen", "zigbee", fields))
	configured, err := LoadConfiguredDevices(filename)
	require.NoError(t, err)
	require.Equal(t, "Cooling", configured[0].Label)
	require.Equal(t, "❄️", configured[0].Icon)
	require.Equal(t, 0, *configured[0].Options.Min)
	require.Equal(t, 200, *configured[0].Options.Max)
	require.True(t, *configured[0].Options.Hidden)
	require.Equal(t, "IR_CODE", (*configured[0].Options.Commands)["cold"])
	fields.Min, fields.Max = nil, nil
	fields.Hidden = ToPtr(false)
	require.NoError(t, UpdateDeviceFields(filename, "remote", "ircontrol", "Kitchen", "zigbee", fields))
	configured, err = LoadConfiguredDevices(filename)
	require.NoError(t, err)
	require.Nil(t, configured[0].Options.Min)
	require.Nil(t, configured[0].Options.Max)
	require.False(t, *configured[0].Options.Hidden)
	require.True(t, configured[0].Options.PreventResending)
}
