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
	s.Icon = "⏰"

	s.Type = "time"
	s.Triggers = []string{"day", "month", "year", "hour", "minute", "second", "weekday"}

	s.values = make(map[string]int)
	s.triggerValues = make(map[string]int)

	s.persistentFields = []string{"day", "month", "year", "hour", "weekday"}

	return &s
}

func (s *DimmyTime) InitRule(rule *Rule) {
	now := time.Now()
	s.UpdateRule(rule, "day", now.Day())
	s.UpdateRule(rule, "month", int(now.Month()))
	s.UpdateRule(rule, "year", now.Year())
	s.UpdateRule(rule, "hour", now.Hour())
	s.UpdateRule(rule, "minute", now.Minute())
	s.UpdateRule(rule, "second", now.Second())
}

func (s *DimmyTime) AddRule(rule *Rule) {
	s.InitRule(rule)
	s.rules = append(s.rules, rule)
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

func (s *DimmyTime) CreateTriggerConfig(trigger string, value int) core.TriggerConfig {
	return core.TriggerConfig{
		DeviceName: "time",
		Key:        trigger,
		Active:     true,
		Condition: core.ReceiverConditionConfig{
			Operator: "==",
			Value:    value,
		},
	}
}

func (s *DimmyTime) CreateTriggersFromTime(value time.Time) []core.TriggerConfig {
	triggers := make([]core.TriggerConfig, 6)
	triggers[0] = s.CreateTriggerConfig("day", value.Day())
	triggers[1] = s.CreateTriggerConfig("month", int(value.Month()))
	triggers[2] = s.CreateTriggerConfig("year", value.Year())
	triggers[3] = s.CreateTriggerConfig("hour", value.Hour())
	triggers[4] = s.CreateTriggerConfig("minute", value.Minute())
	triggers[5] = s.CreateTriggerConfig("second", value.Second())

	return triggers
}
