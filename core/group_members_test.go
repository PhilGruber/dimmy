package core

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateGroupMembers(t *testing.T) {
	devices := []DeviceConfig{
		{Name: "group", Type: "group"},
		{Name: "lamp", Type: "light"},
		{Name: "zigbee-lamp", Type: "zlight"},
		{Name: "sensor", Type: "device"},
		{Name: "other-group", Type: "group"},
	}
	for _, tc := range []struct {
		name    string
		members []string
		valid   bool
	}{
		{"empty", []string{}, true},
		{"one", []string{"lamp"}, true},
		{"lights", []string{"lamp", "zigbee-lamp"}, true},
		{"different type", []string{"lamp", "sensor"}, false},
		{"unknown", []string{"missing"}, false},
		{"self", []string{"group"}, false},
		{"duplicate", []string{"lamp", "lamp"}, false},
		{"nested group", []string{"other-group"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateGroupMembers("group", tc.members, devices)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
