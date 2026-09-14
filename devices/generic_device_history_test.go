package devices

import (
	"testing"
	"time"

	"github.com/PhilGruber/dimmy/core"
	"github.com/stretchr/testify/require"
)

type recordedHistory struct {
	device    string
	sensor    string
	value     any
	timestamp time.Time
}

func (h *recordedHistory) AddSensorHistory(device, sensor string, value any, timestamp time.Time) error {
	h.device = device
	h.sensor = sensor
	h.value = value
	h.timestamp = timestamp
	return nil
}

func TestGenericDeviceAddHistoryPersistsAndRetainsMemoryHistory(t *testing.T) {
	history := &recordedHistory{}
	sensors := []core.Sensor{{Name: "temperature"}}
	d := NewDevice(core.DeviceConfig{
		Name: "Kitchen Sensor",
		Options: &core.ConfigOptions{
			Sensors: &sensors,
		},
	}, history)

	d.AddHistory("temperature", 21.5)

	require.Len(t, d.Values["temperature"].History, 1)
	require.Equal(t, 21.5, d.Values["temperature"].History[0].Value)
	require.Equal(t, "Kitchen Sensor", history.device)
	require.Equal(t, "temperature", history.sensor)
	require.Equal(t, 21.5, history.value)
	require.False(t, history.timestamp.IsZero())
}
