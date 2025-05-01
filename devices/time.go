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

func NewDimmyTime(config core.DeviceConfig) *DimmyTime {
	s := DimmyTime{}
	s.setBaseConfig(config)
	s.Hidden = true

	s.Type = "time"
	s.Triggers = []string{"day", "month", "year", "hour", "minute", "second", "weekday"}

	s.values = make(map[string]int)
	s.triggerValues = make(map[string]int)

	s.persistentFields = []string{"day", "month", "year", "hour", "weekday"}

	return &s
}

func (s *DimmyTime) UpdateValue() (float64, bool) {
	now := time.Now()
	if s.values["day"] != now.Day() {
		s.UpdateRules("day", now.Day())
	}
	if s.values["month"] != int(now.Month()) {
		s.UpdateRules("month", int(now.Month()))
	}
	if s.values["year"] != now.Year() {
		s.UpdateRules("year", now.Year())
	}
	if s.values["hour"] != now.Hour() {
		s.UpdateRules("hour", now.Hour())
	}
	if s.values["minute"] != now.Minute() {
		s.UpdateRules("minute", now.Minute())
	}
	if s.values["second"] != now.Second() {
		s.UpdateRules("second", now.Second())
	}
	if s.values["weekday"] != int(now.Weekday()) {
		s.UpdateRules("weekday", int(now.Weekday()))
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

func (s *DimmyTime) ClearTrigger(trigger string) {
	if trigger == "minute" || trigger == "second" {
		s.triggerValues[trigger] = -1
	}
}

func (s *DimmyTime) CreateTrigger(trigger string, value int) Trigger {
	c := condition{Operator: "==", Value: value}
	return Trigger{Device: s, Key: trigger, Condition: &c}
}

func (s *DimmyTime) CreateTriggerFromTime(value time.Time) []Trigger {
	triggers := make([]Trigger, 6)
	triggers[0] = s.CreateTrigger("day", value.Day())
	triggers[1] = s.CreateTrigger("month", int(value.Month()))
	triggers[2] = s.CreateTrigger("year", value.Year())
	triggers[3] = s.CreateTrigger("hour", value.Hour())
	triggers[4] = s.CreateTrigger("minute", value.Minute())
	triggers[5] = s.CreateTrigger("second", value.Second())

	return triggers
}
