package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestAddDeviceToConfig(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "dimmyd.conf.yaml")
	err := os.WriteFile(filename, []byte("mqtt_server: localhost\ncustom_setting: keep-me\ndevices:\n  - name: existing\n    type: sensor\n    topic: sensors/existing\n"), 0o640)
	require.NoError(t, err)

	sensors := []Sensor{{Name: "temperature", Icon: "temp"}}
	err = AddDeviceToConfig(filename, DeviceConfig{
		Name:  "Kitchen Sensor",
		Type:  "device",
		Topic: "zigbee/kitchen",
		Options: &ConfigOptions{
			Sensors: &sensors,
		},
	})
	require.NoError(t, err)

	data, err := os.ReadFile(filename)
	require.NoError(t, err)

	var config struct {
		CustomSetting string         `yaml:"custom_setting"`
		Devices       []DeviceConfig `yaml:"devices"`
	}
	require.NoError(t, yaml.Unmarshal(data, &config))
	require.Equal(t, "keep-me", config.CustomSetting)
	require.Len(t, config.Devices, 2)
	require.Equal(t, "Kitchen Sensor", config.Devices[1].Name)
	require.Equal(t, "zigbee/kitchen", config.Devices[1].Topic)
	require.Equal(t, "temperature", (*config.Devices[1].Options.Sensors)[0].Name)

	info, err := os.Stat(filename)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o640), info.Mode().Perm())
}
