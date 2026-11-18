package core

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestCommandsPreserveOrderAndLookup(t *testing.T) {
	var commands Commands
	require.NoError(t, yaml.Unmarshal([]byte("power: IR_POWER\ncool: IR_COOL\noff: IR_OFF\n"), &commands))
	require.Equal(t, []string{"power", "cool", "off"}, commands.Names())
	value, ok := commands.Get("cool")
	require.True(t, ok)
	require.Equal(t, "IR_COOL", value)
	_, ok = commands.Get("missing")
	require.False(t, ok)
	data, err := yaml.Marshal(commands)
	require.NoError(t, err)
	var restored Commands
	require.NoError(t, yaml.Unmarshal(data, &restored))
	require.Equal(t, commands, restored)
}

func TestCommandsRejectInvalidYAML(t *testing.T) {
	for _, input := range []string{"power: first\npower: second\n", "[power, cool]", "power: [not, a, code]"} {
		var commands Commands
		require.Error(t, yaml.Unmarshal([]byte(input), &commands))
	}
}
