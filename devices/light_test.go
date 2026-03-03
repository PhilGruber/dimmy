package devices

import (
	"encoding/json"
	"testing"

	"github.com/PhilGruber/dimmy/core"
)

func TestNewLight_MqttStateTopicTransform(t *testing.T) {
	l := NewLight(core.DeviceConfig{
		Name:  "test",
		Topic: "cmnd/living/dimmer",
	})
	if l.MqttState != "tele/living/STATE" {
		t.Errorf("expected tele/living/STATE, got %s", l.MqttState)
	}
}

func TestNewLight_NonCmndTopicUnchanged(t *testing.T) {
	l := NewLight(core.DeviceConfig{
		Name:  "test",
		Topic: "zigbee2mqtt/light1",
	})
	if l.MqttState != "zigbee2mqtt/light1" {
		t.Errorf("expected MqttState=zigbee2mqtt/light1, got %s", l.MqttState)
	}
}

func TestLight_PercentageToValue_Zero(t *testing.T) {
	l := newTestLight()
	if v := l.PercentageToValue(0); v != 0 {
		t.Errorf("expected 0, got %d", v)
	}
}

func TestLight_PercentageToValue_One(t *testing.T) {
	l := newTestLight()
	if v := l.PercentageToValue(1); v != 1 {
		t.Errorf("expected 1, got %d", v)
	}
}

func TestLight_PercentageToValue_Hundred(t *testing.T) {
	l := newTestLight()
	if v := l.PercentageToValue(100); v != 100 {
		t.Errorf("expected 100, got %d", v)
	}
}

func TestLight_PercentageToValue_Middle(t *testing.T) {
	l := newTestLight()
	// Min=0,Max=100: 1 + (100-0-1)*(50-1)/99 = 1+49 = 50
	if v := l.PercentageToValue(50); v != 50 {
		t.Errorf("expected 50, got %d", v)
	}
}

func TestLight_ValueToPercentage_AtMin(t *testing.T) {
	l := newTestLight()
	if p := l.ValueToPercentage(0); p != 0 {
		t.Errorf("expected 0, got %f", p)
	}
}

func TestLight_ValueToPercentage_MinPlusOne(t *testing.T) {
	l := newTestLight()
	// value=1 = Min+1 → 1%
	if p := l.ValueToPercentage(1); p != 1 {
		t.Errorf("expected 1, got %f", p)
	}
}

func TestLight_ValueToPercentage_AtMax(t *testing.T) {
	l := newTestLight()
	if p := l.ValueToPercentage(100); p != 100 {
		t.Errorf("expected 100, got %f", p)
	}
}

func TestLight_ValueToPercentage_Middle(t *testing.T) {
	l := newTestLight()
	// value=50: 1 + (50-0-1)*99/(100-0-1) = 1+49 = 50
	if p := l.ValueToPercentage(50); p != 50 {
		t.Errorf("expected 50, got %f", p)
	}
}

func TestLight_GetMessageHandler_NumericPayload(t *testing.T) {
	l := newTestLight()
	// Target=0, Current=0 → handler proceeds
	l.GetMessageHandler(nil, l)(nil, &mockMessage{payload: []byte("50")})
	if l.GetCurrent() != 50 {
		t.Errorf("expected Current=50, got %f", l.GetCurrent())
	}
}

func TestLight_GetMessageHandler_JSONPayloadOn(t *testing.T) {
	l := newTestLight()
	payload, _ := json.Marshal(lightStateMessage{Value: 50, State: "ON"})
	l.GetMessageHandler(nil, l)(nil, &mockMessage{payload: payload})
	if l.GetCurrent() != 50 {
		t.Errorf("expected Current=50, got %f", l.GetCurrent())
	}
}

func TestLight_GetMessageHandler_JSONPayloadOff(t *testing.T) {
	l := newTestLight()
	handler := l.GetMessageHandler(nil, l)
	// Drive current to 50 so Target==Current for second handler call.
	payload, _ := json.Marshal(lightStateMessage{Value: 50, State: "ON"})
	handler(nil, &mockMessage{payload: payload})
	// Now send OFF — value is forced to 0.
	payload, _ = json.Marshal(lightStateMessage{Value: 50, State: "OFF"})
	handler(nil, &mockMessage{payload: payload})
	if l.GetCurrent() != 0 {
		t.Errorf("expected Current=0 when POWER=OFF, got %f", l.GetCurrent())
	}
}

func TestLight_GetMessageHandler_InvalidPayload(t *testing.T) {
	l := newTestLight()
	handler := l.GetMessageHandler(nil, l)
	// Bring current to 50 first.
	payload, _ := json.Marshal(lightStateMessage{Value: 50, State: "ON"})
	handler(nil, &mockMessage{payload: payload})
	// Invalid payload — handler returns early, current stays 50.
	handler(nil, &mockMessage{payload: []byte("not-valid")})
	if l.GetCurrent() != 50 {
		t.Errorf("expected Current=50 (unchanged), got %f", l.GetCurrent())
	}
}

func TestLight_GetMessageHandler_IgnoredWhileMoving(t *testing.T) {
	l := newTestLight()
	// ProcessRequest sets Target=75 but leaves Current=0 → light is "moving".
	l.ProcessRequest(core.SwitchRequest{Value: "75"})
	l.GetMessageHandler(nil, l)(nil, &mockMessage{payload: []byte("50")})
	// Handler skips because GetTarget() != round(GetCurrent()).
	if l.GetCurrent() != 0 {
		t.Errorf("expected Current=0 (ignored while moving), got %f", l.GetCurrent())
	}
}

func TestLight_SetReceiverValue_Brightness(t *testing.T) {
	l := newTestLight()
	l.SetReceiverValue("brightness", float64(60))
	if l.GetTarget() != 60 {
		t.Errorf("expected Target=60, got %f", l.GetTarget())
	}
}
