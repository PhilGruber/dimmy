package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DefaultDatabasePath is used when database_path is not set in dimmyd's
// configuration file.
const DefaultDatabasePath = "/var/lib/dimmy/dimmy.db"

// SensorHistoryStore is the subset of the history database used by devices.
// It keeps the device package independent of database/sql.
type SensorHistoryStore interface {
	AddSensorHistory(device, sensor string, value any, timestamp time.Time) error
}

type TimeValue struct {
	Time  time.Time
	Value string
}

// OpenHistoryDatabase opens (and creates, when necessary) the SQLite database
// and ensures its sensor history table exists.
func OpenHistoryDatabase(path string) (*HistoryDatabase, error) {
	if path == "" {
		path = DefaultDatabasePath
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open history database: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS sensor_history (
		id INTEGER PRIMARY KEY,
		device TEXT NOT NULL,
		sensor TEXT NOT NULL,
		value ANY,
		timestamp DATETIME NOT NULL
	)`); err != nil {
		db.Close()
		return nil, fmt.Errorf("create sensor_history table: %w", err)
	}
	return &HistoryDatabase{DB: db}, nil
}

// HistoryDatabase persists sensor readings in SQLite.
type HistoryDatabase struct {
	DB *sql.DB
}

func (d *HistoryDatabase) Close() error {
	return d.DB.Close()
}

func (d *HistoryDatabase) AddSensorHistory(device, sensor string, value any, timestamp time.Time) error {
	value, err := sqliteValue(value)
	if err != nil {
		return fmt.Errorf("encode sensor value: %w", err)
	}
	_, err = d.DB.Exec(
		`INSERT INTO sensor_history (device, sensor, value, timestamp) VALUES (?, ?, ?, ?)`,
		device, sensor, value, timestamp,
	)
	if err != nil {
		return fmt.Errorf("insert sensor history: %w", err)
	}
	return nil
}

func (d *HistoryDatabase) GetSensorHistory(device string, sensor string) ([]TimeValue, error) {
	rows, err := d.DB.QueryContext(
		context.Background(),
		`SELECT timestamp, value FROM sensor_history WHERE device=? AND sensor=? ORDER BY timestamp DESC`,
		device, sensor)
	defer rows.Close()
	if err != nil {
		return nil, err
	}
	result := make([]TimeValue, 0)
	for rows.Next() {
		var timestamp time.Time
		var value string
		if err := rows.Scan(&timestamp, &value); err != nil {
			return nil, err
		}
		result = append(result, TimeValue{timestamp, value})
	}
	return result, nil
}

func sqliteValue(value any) (any, error) {
	switch value.(type) {
	case nil, bool, string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, []byte:
		return value, nil
	default:
		return json.Marshal(value)
	}
}
