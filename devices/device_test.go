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
		"battery_low",
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
		"state",
		"child_lock",
	} {
		if !d.likelyControl(field) {
			t.Errorf("expected %q to be treated as a likely control field", field)
		}
	}
}
