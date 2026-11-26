package core

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestRuleControlDelay(t *testing.T) {
	var rule RuleConfig
	require.NoError(t, yaml.Unmarshal([]byte("receivers:\n  - device: lamp\n    key: brightness\n    value: '0'\n    delay: 600\n"), &rule))
	require.Equal(t, 600, rule.Receivers[0].Delay)
	require.NoError(t, rule.Validate())
	data, err := yaml.Marshal(rule)
	require.NoError(t, err)
	require.Contains(t, string(data), "delay: 600")
	rule.Receivers[0].Delay = -1
	require.Error(t, rule.Validate())
}
