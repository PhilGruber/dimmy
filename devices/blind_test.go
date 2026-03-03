package devices

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/PhilGruber/dimmy/core"
	"gopkg.in/yaml.v3"
)

// mockMessage implements mqtt.Message for testing GetMessageHandler.
type mockMessage struct {
	payload []byte
}

func (m *mockMessage) Duplicate() bool    { return false }
func (m *mockMessage) Qos() byte         { return 0 }
func (m *mockMessage) Retained() bool    { return false }
func (m *mockMessage) Topic() string     { return "" }
func (m *mockMessage) MessageID() uint16 { return 0 }
func (m *mockMessage) Payload() []byte   { return m.payload }
func (m *mockMessage) Ack()              {}

func defaultBlindConfig() core.DeviceConfig {
	return core.DeviceConfig{
		Name:  "test-blind",
		Topic: "zigbee2mqtt/blind1",
	}
}

func blindConfigWithMinMax(t *testing.T, min, max int) core.DeviceConfig {
	t.Helper()
	var cfg core.DeviceConfig
	raw := fmt.Sprintf("name: custom-blind\ntopic: zigbee2mqtt/blind2\noptions:\n  min: %d\n  max: %d\n", min, max)
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}
	return cfg
}

func TestNewBlind_Defaults(t *testing.T) {
	b := NewBlind(defaultBlindConfig())

	if b.Min != 0 {
		t.Errorf("expected Min=0, got %d", b.Min)
	}
	if b.Max != 100 {
		t.Errorf("expected Max=100, got %d", b.Max)
	}
	if b.Type != "blind" {
		t.Errorf("expected Type=blind, got %s", b.Type)
	}
	if b.Name != "test-blind" {
		t.Errorf("expected Name=test-blind, got %s", b.Name)
	}
	if b.MqttTopic != "zigbee2mqtt/blind1" {
		t.Errorf("expected MqttTopic=zigbee2mqtt/blind1, got %s", b.MqttTopic)
	}
	if b.MqttState != "zigbee2mqtt/blind1" {
		t.Errorf("expected MqttState=zigbee2mqtt/blind1, got %s", b.MqttState)
	}
	if len(b.Receivers) != 1 || b.Receivers[0] != "position" {
		t.Errorf("expected Receivers=[position], got %v", b.Receivers)
	}
	if b.needsSending {
		t.Error("expected needsSending=false after construction")
	}
}

func TestNewBlind_CustomMinMax(t *testing.T) {
	cfg := blindConfigWithMinMax(t, 2, 8) // single-digit so rune arithmetic works
	b := NewBlind(cfg)

	if b.Min != 2 {
		t.Errorf("expected Min=2, got %d", b.Min)
	}
	if b.Max != 8 {
		t.Errorf("expected Max=8, got %d", b.Max)
	}
}

func TestProcessRequest_Absolute(t *testing.T) {
	b := NewBlind(defaultBlindConfig())

	b.ProcessRequest(core.SwitchRequest{Value: "50"})

	if b.Position != 50 {
		t.Errorf("expected Position=50, got %d", b.Position)
	}
	if !b.needsSending {
		t.Error("expected needsSending=true after ProcessRequest")
	}
}

func TestProcessRequest_ClampToMax(t *testing.T) {
	b := NewBlind(defaultBlindConfig())

	b.ProcessRequest(core.SwitchRequest{Value: "150"})

	if b.Position != 100 {
		t.Errorf("expected Position clamped to 100, got %d", b.Position)
	}
}

func TestProcessRequest_ClampToMin(t *testing.T) {
	b := NewBlind(defaultBlindConfig())

	b.ProcessRequest(core.SwitchRequest{Value: "-50"})

	if b.Position != 0 {
		t.Errorf("expected Position clamped to 0, got %d", b.Position)
	}
}

func TestProcessRequest_RelativePositive(t *testing.T) {
	b := NewBlind(defaultBlindConfig())
	b.Position = 40

	b.ProcessRequest(core.SwitchRequest{Value: "+20"})

	if b.Position != 60 {
		t.Errorf("expected Position=60, got %d", b.Position)
	}
}

func TestProcessRequest_RelativeNegative(t *testing.T) {
	b := NewBlind(defaultBlindConfig())
	b.Position = 40

	b.ProcessRequest(core.SwitchRequest{Value: "-15"})

	if b.Position != 25 {
		t.Errorf("expected Position=25, got %d", b.Position)
	}
}

func TestProcessRequest_RelativeClampedAtMax(t *testing.T) {
	b := NewBlind(defaultBlindConfig())
	b.Position = 90

	b.ProcessRequest(core.SwitchRequest{Value: "+20"})

	if b.Position != 100 {
		t.Errorf("expected Position clamped to 100, got %d", b.Position)
	}
}

func TestProcessRequest_RelativeClampedAtMin(t *testing.T) {
	b := NewBlind(defaultBlindConfig())
	b.Position = 5

	b.ProcessRequest(core.SwitchRequest{Value: "-20"})

	if b.Position != 0 {
		t.Errorf("expected Position clamped to 0, got %d", b.Position)
	}
}

func TestUpdateValue_NoPendingChange(t *testing.T) {
	b := NewBlind(defaultBlindConfig())

	val, send := b.UpdateValue()

	if send {
		t.Error("expected send=false when no pending change")
	}
	if val != 0 {
		t.Errorf("expected val=0 when no pending change, got %f", val)
	}
}

func TestUpdateValue_PendingChange(t *testing.T) {
	b := NewBlind(defaultBlindConfig())
	b.ProcessRequest(core.SwitchRequest{Value: "75"})

	val, send := b.UpdateValue()

	if !send {
		t.Error("expected send=true after ProcessRequest")
	}
	if val != 75 {
		t.Errorf("expected val=75, got %f", val)
	}
}

func marshalBlindPayload(t *testing.T, msg core.Zigbee2MqttBlindStatusMessage) []byte {
	t.Helper()
	b, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	return b
}

func TestGetMessageHandler_UpdatesPosition(t *testing.T) {
	b := NewBlind(defaultBlindConfig())
	pos := 42
	payload := marshalBlindPayload(t, core.Zigbee2MqttBlindStatusMessage{
		Zigbee2MqttBlindMessage: core.Zigbee2MqttBlindMessage{Position: &pos},
	})

	b.GetMessageHandler(nil, b)(nil, &mockMessage{payload: payload})

	if b.Position != 42 {
		t.Errorf("expected Position=42, got %d", b.Position)
	}
}

func TestGetMessageHandler_StateClose(t *testing.T) {
	b := NewBlind(defaultBlindConfig())
	b.Position = 50
	pos := 0
	state := "CLOSE"
	payload := marshalBlindPayload(t, core.Zigbee2MqttBlindStatusMessage{
		Zigbee2MqttBlindMessage: core.Zigbee2MqttBlindMessage{Position: &pos},
		State:                   &state,
	})

	b.GetMessageHandler(nil, b)(nil, &mockMessage{payload: payload})

	if b.GetCurrent() != 0 {
		t.Errorf("expected Current=0 on CLOSE, got %f", b.GetCurrent())
	}
}

func TestGetMessageHandler_StateOpen(t *testing.T) {
	b := NewBlind(defaultBlindConfig())
	pos := 100
	state := "OPEN"
	payload := marshalBlindPayload(t, core.Zigbee2MqttBlindStatusMessage{
		Zigbee2MqttBlindMessage: core.Zigbee2MqttBlindMessage{Position: &pos},
		State:                   &state,
	})

	b.GetMessageHandler(nil, b)(nil, &mockMessage{payload: payload})

	if b.GetCurrent() != 100 {
		t.Errorf("expected Current=100 on OPEN, got %f", b.GetCurrent())
	}
}

func TestGetMessageHandler_Battery(t *testing.T) {
	b := NewBlind(defaultBlindConfig())
	pos := 50
	batt := 80
	payload := marshalBlindPayload(t, core.Zigbee2MqttBlindStatusMessage{
		Zigbee2MqttMessage:      core.Zigbee2MqttMessage{Battery: &batt},
		Zigbee2MqttBlindMessage: core.Zigbee2MqttBlindMessage{Position: &pos},
	})

	b.GetMessageHandler(nil, b)(nil, &mockMessage{payload: payload})

	if b.Battery == nil || *b.Battery != 80 {
		t.Errorf("expected Battery=80, got %v", b.Battery)
	}
}

func TestGetMessageHandler_LinkQuality(t *testing.T) {
	b := NewBlind(defaultBlindConfig())
	pos := 50
	lq := 120
	payload := marshalBlindPayload(t, core.Zigbee2MqttBlindStatusMessage{
		Zigbee2MqttMessage:      core.Zigbee2MqttMessage{LinkQuality: &lq},
		Zigbee2MqttBlindMessage: core.Zigbee2MqttBlindMessage{Position: &pos},
	})

	b.GetMessageHandler(nil, b)(nil, &mockMessage{payload: payload})

	if b.LinkQuality == nil || *b.LinkQuality != 120 {
		t.Errorf("expected LinkQuality=120, got %v", b.LinkQuality)
	}
}

func TestGetMessageHandler_InvalidJSON(t *testing.T) {
	b := NewBlind(defaultBlindConfig())
	b.Position = 33

	b.GetMessageHandler(nil, b)(nil, &mockMessage{payload: []byte("not-json")})

	if b.Position != 33 {
		t.Errorf("expected Position unchanged (33), got %d", b.Position)
	}
}

func TestGetMessageHandler_MissingPosition(t *testing.T) {
	b := NewBlind(defaultBlindConfig())
	b.Position = 55
	// Valid JSON but no position field — handler returns early without updating.
	payload := marshalBlindPayload(t, core.Zigbee2MqttBlindStatusMessage{})

	b.GetMessageHandler(nil, b)(nil, &mockMessage{payload: payload})

	if b.Position != 55 {
		t.Errorf("expected Position unchanged (55), got %d", b.Position)
	}
}
