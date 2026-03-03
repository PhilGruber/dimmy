package devices

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/PhilGruber/dimmy/core"
	"gopkg.in/yaml.v3"
)

// mockDevice is a minimal DeviceInterface implementation for group tests.
// It records every ProcessRequest call and returns a fixed current value.
type mockDevice struct {
	Device
	deviceType string
	current    float64
	requests   []core.SwitchRequest
}

func newMockDevice(name, deviceType string, current float64) *mockDevice {
	d := &mockDevice{deviceType: deviceType, current: current}
	d.setBaseConfig(core.DeviceConfig{Name: name, Topic: "mock/" + name})
	return d
}

func (m *mockDevice) GetType() string                      { return m.deviceType }
func (m *mockDevice) GetCurrent() float64                  { return m.current }
func (m *mockDevice) UpdateValue() (float64, bool)         { return m.current, false }
func (m *mockDevice) ProcessRequest(r core.SwitchRequest)  { m.requests = append(m.requests, r) }

// groupConfig builds a DeviceConfig that lists the given device names under options.devices.
func groupConfig(t *testing.T, deviceNames []string) core.DeviceConfig {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("name: test-group\ntopic: group/topic\noptions:\n  devices:\n")
	for _, name := range deviceNames {
		fmt.Fprintf(&sb, "    - %s\n", name)
	}
	var cfg core.DeviceConfig
	if err := yaml.Unmarshal([]byte(sb.String()), &cfg); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}
	return cfg
}

func TestNewGroup_NilOptions_ReturnsNil(t *testing.T) {
	cfg := core.DeviceConfig{Name: "test-group"}
	g := NewGroup(cfg, map[string]DeviceInterface{})
	if g != nil {
		t.Error("expected nil when Options is nil")
	}
}

func TestNewGroup_NoDevices_ReturnsNil(t *testing.T) {
	// Options is present but has no devices key.
	var cfg core.DeviceConfig
	yaml.Unmarshal([]byte("name: test-group\noptions:\n  hidden: false\n"), &cfg)
	g := NewGroup(cfg, map[string]DeviceInterface{})
	if g != nil {
		t.Error("expected nil when options.devices is absent")
	}
}

func TestNewGroup_MixedDeviceTypes_ReturnsNil(t *testing.T) {
	allDevices := map[string]DeviceInterface{
		"dev1": newMockDevice("dev1", "light", 0),
		"dev2": newMockDevice("dev2", "blind", 0),
	}
	g := NewGroup(groupConfig(t, []string{"dev1", "dev2"}), allDevices)
	if g != nil {
		t.Error("expected nil when devices have different types")
	}
}

func TestNewGroup_Success(t *testing.T) {
	allDevices := map[string]DeviceInterface{
		"dev1": newMockDevice("dev1", "light", 50),
		"dev2": newMockDevice("dev2", "light", 75),
	}
	g := NewGroup(groupConfig(t, []string{"dev1", "dev2"}), allDevices)
	if g == nil {
		t.Fatal("expected non-nil group")
	}
	if g.Type != "light" {
		t.Errorf("expected Type=light, got %s", g.Type)
	}
	if len(g.devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(g.devices))
	}
}

func TestGroup_GetCurrent_ReturnsMaxOfDevices(t *testing.T) {
	allDevices := map[string]DeviceInterface{
		"dev1": newMockDevice("dev1", "light", 50),
		"dev2": newMockDevice("dev2", "light", 75),
	}
	g := NewGroup(groupConfig(t, []string{"dev1", "dev2"}), allDevices)
	if g.GetCurrent() != 75 {
		t.Errorf("expected GetCurrent=75, got %f", g.GetCurrent())
	}
}

func TestGroup_ProcessRequest_AbsoluteDelegatesToAll(t *testing.T) {
	dev1 := newMockDevice("dev1", "light", 50)
	dev2 := newMockDevice("dev2", "light", 75)
	allDevices := map[string]DeviceInterface{"dev1": dev1, "dev2": dev2}
	g := NewGroup(groupConfig(t, []string{"dev1", "dev2"}), allDevices)

	g.ProcessRequest(core.SwitchRequest{Value: "60"})

	if len(dev1.requests) == 0 {
		t.Fatal("expected dev1 to receive a request")
	}
	if dev1.requests[0].Value != "60" {
		t.Errorf("expected dev1 Value=60, got %s", dev1.requests[0].Value)
	}
	if len(dev2.requests) == 0 {
		t.Fatal("expected dev2 to receive a request")
	}
	if dev2.requests[0].Value != "60" {
		t.Errorf("expected dev2 Value=60, got %s", dev2.requests[0].Value)
	}
}

func TestGroup_ProcessRequest_RelativeResolvesBeforeDelegating(t *testing.T) {
	dev1 := newMockDevice("dev1", "light", 50)
	dev2 := newMockDevice("dev2", "light", 75)
	allDevices := map[string]DeviceInterface{"dev1": dev1, "dev2": dev2}
	g := NewGroup(groupConfig(t, []string{"dev1", "dev2"}), allDevices)

	// GetCurrent() = max(50, 75) = 75; +10 → 85
	g.ProcessRequest(core.SwitchRequest{Value: "+10"})

	if len(dev1.requests) == 0 {
		t.Fatal("expected dev1 to receive a request")
	}
	val, err := strconv.ParseFloat(dev1.requests[0].Value, 64)
	if err != nil {
		t.Fatalf("dev1 request value not parseable: %s", dev1.requests[0].Value)
	}
	if val != 85 {
		t.Errorf("expected resolved value=85, got %f", val)
	}
}
