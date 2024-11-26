package devices

import (
	"github.com/PhilGruber/dimmy/core"
	"time"
)

type DimmyTime struct {
	Device

	values        map[string]int
	triggerValues map[string]int
}

func MakeDimmyTime(config core.DeviceConfig) DimmyTime {
	s := DimmyTime{}
	s.setBaseConfig(config)
	s.Hidden = true

	s.Type = "time"
	s.Triggers = []string{"day", "month", "year", "hour", "minute", "second", "weekday"}

	s.values = make(map[string]int)
	s.triggerValues = make(map[string]int)

	return s
}

func NewDimmyTime(config core.DeviceConfig) *DimmyTime {
	s := MakeDimmyTime(config)
	return &s
}

func (s *DimmyTime) UpdateValue() (float64, bool) {
	now := time.Now()
	if s.values["day"] != now.Day() {
		s.triggerValues["day"] = now.Day()
	}
	if s.values["month"] != int(now.Month()) {
		s.triggerValues["month"] = int(now.Month())
	}
	if s.values["year"] != now.Year() {
		s.triggerValues["year"] = now.Year()
	}
	if s.values["hour"] != now.Hour() {
		s.triggerValues["hour"] = now.Hour()
	}
	if s.values["minute"] != now.Minute() {
		s.triggerValues["minute"] = now.Minute()
	}
	if s.values["second"] != now.Second() {
		s.triggerValues["second"] = now.Second()
	}
	if s.values["weekday"] != int(now.Weekday()) {
		s.triggerValues["weekday"] = int(now.Weekday())
	}
	s.values["day"] = now.Day()
	s.values["month"] = int(now.Month())
	s.values["year"] = now.Year()
	s.values["hour"] = now.Hour()
	s.values["minute"] = now.Minute()
	s.values["second"] = now.Second()
	s.values["weekday"] = int(now.Weekday())

	return 0, false
}

func (s *DimmyTime) GetTriggerValue(trigger string) interface{} {
	if value, ok := s.triggerValues[trigger]; ok {
		return value
	}
	return nil
}

func (s *DimmyTime) ClearTrigger(trigger string) {
	if trigger == "minute" || trigger == "second" {
		s.triggerValues[trigger] = -1
	}
}
