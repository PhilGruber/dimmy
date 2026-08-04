package devices

import "testing"

func TestDevice_LikelySensor_CommonZigbeeFields(t *testing.T) {
	d := &Device{}

	for _, field := range []string{
		"temperature",
		"humidity",
		"pressure",
		"co2",
		"voc",
		"pm25",
		"smoke",
		"gas",
		"water_leak",
		"tamper",
		"power",
	} {
		if !d.likelySensor(field) {
			t.Errorf("expected %q to be treated as a likely sensor field", field)
		}
	}
}

func TestDevice_LikelyControl_CommonZigbeeFields(t *testing.T) {
	d := &Device{}

	for _, field := range []string{
		"brightness",
		"color",
		"color_temp",
		"effect",
		"transition",
		"fan_mode",
		"power_on_behavior",
		"tilt",
		"child_lock",
	} {
		if !d.likelyControl(field) {
			t.Errorf("expected %q to be treated as a likely control field", field)
		}
	}
}

type mockMessage struct {
	payload []byte
}

func (m *mockMessage) Duplicate() bool   { return false }
func (m *mockMessage) Qos() byte         { return 0 }
func (m *mockMessage) Retained() bool    { return false }
func (m *mockMessage) Topic() string     { return "" }
func (m *mockMessage) MessageID() uint16 { return 0 }
func (m *mockMessage) Payload() []byte   { return m.payload }
func (m *mockMessage) Ack()              {}
