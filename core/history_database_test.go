package core

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHistoryDatabaseStoresSensorReading(t *testing.T) {
	db, err := OpenHistoryDatabase(filepath.Join(t.TempDir(), "history.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	timestamp := time.Date(2026, time.September, 14, 12, 0, 0, 0, time.UTC)
	require.NoError(t, db.AddSensorHistory("Kitchen", "temperature", 21.5, timestamp))

	var id int
	var device, sensor string
	var value float64
	var storedTimestamp time.Time
	require.NoError(t, db.DB.QueryRow(`SELECT id, device, sensor, value, timestamp FROM sensor_history`).Scan(
		&id, &device, &sensor, &value, &storedTimestamp,
	))
	require.Positive(t, id)
	require.Equal(t, "Kitchen", device)
	require.Equal(t, "temperature", sensor)
	require.Equal(t, 21.5, value)
	require.True(t, timestamp.Equal(storedTimestamp))
}

func TestHistoryDatabaseEncodesCompositeValues(t *testing.T) {
	db, err := OpenHistoryDatabase(filepath.Join(t.TempDir(), "history.db"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	require.NoError(t, db.AddSensorHistory("Kitchen", "state", map[string]any{"open": true}, time.Now()))

	var value string
	require.NoError(t, db.DB.QueryRow(`SELECT value FROM sensor_history`).Scan(&value))
	require.JSONEq(t, `{"open":true}`, value)
}
