package devices

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/PhilGruber/dimmy/core"
	"github.com/stretchr/testify/require"
)

func TestGenericDeviceRestoresLatestHistoryReading(t *testing.T) {
	db, err := core.OpenHistoryDatabase(filepath.Join(t.TempDir(), "history.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	start := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	for i := 11; i >= 0; i-- {
		require.NoError(t, db.AddSensorHistory("Kitchen", "temperature", float64(i)+.5, start.Add(time.Duration(i)*time.Minute)))
	}
	require.NoError(t, db.AddSensorHistory("Other", "temperature", 99, start.Add(time.Hour)))
	sensors := []core.Sensor{{Name: "temperature"}, {Name: "humidity"}}
	config := core.DeviceConfig{Name: "Kitchen", Options: &core.ConfigOptions{Sensors: &sensors}}
	d := NewDevice(config, db)
	value := d.Values["temperature"]
	require.Equal(t, 11.5, value.Value)
	require.True(t, start.Add(11*time.Minute).Equal(value.LastChanged))
	require.Len(t, value.History, 10)
	require.Equal(t, 2.5, value.History[0].Value)
	require.Equal(t, 11.5, value.History[9].Value)
	require.Nil(t, d.Values["humidity"].Value)
	require.True(t, time.Unix(0, 0).Equal(d.Values["humidity"].LastChanged))
	var count int
	require.NoError(t, db.DB.QueryRow(`SELECT COUNT(*) FROM sensor_history WHERE device='Kitchen'`).Scan(&count))
	require.Equal(t, 12, count, "restoration must not persist duplicate readings")
	config.Options.History = core.ToPtr(false)
	d = NewDevice(config, db)
	require.Nil(t, d.Values["temperature"].Value)
	require.Empty(t, d.Values["temperature"].History)
}

func TestGenericDeviceHistoryRestoreFailureKeepsDefault(t *testing.T) {
	db, err := core.OpenHistoryDatabase(filepath.Join(t.TempDir(), "history.db"))
	require.NoError(t, err)
	require.NoError(t, db.Close())
	sensors := []core.Sensor{{Name: "temperature"}}
	d := NewDevice(core.DeviceConfig{Name: "Kitchen", Options: &core.ConfigOptions{Sensors: &sensors}}, db)
	require.Nil(t, d.Values["temperature"].Value)
	require.Empty(t, d.Values["temperature"].History)
}

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
