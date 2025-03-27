package devices

import (
	"fmt"
	"github.com/PhilGruber/dimmy/core"
	"log"
	"reflect"
	"strconv"
	"time"
)

type Rule struct {
	Triggers  []Trigger
	Receivers []Receiver
}

type Trigger struct {
	Device    DeviceInterface
	Key       string
	Condition *condition
}

type condition struct {
	Operator    string
	Value       any
	LastValue   any
	LastChanged *time.Time
}

func (c *condition) Clear() {
	fmt.Printf("Clearing condition %p\n", c)
	c.LastValue = nil
	c.LastChanged = nil
}

func (c *condition) check() bool {
	value, target, err := makeComparable(c.LastValue, c.Value)
	if err != nil {
		log.Println(err)
		return false
	}
	if value == nil || target == nil {
		return false
	}

	switch c.Operator {
	case "==":
		return value == target
	case "!=":
		return value != target
	case ">":
		switch value.(type) {
		case int:
			return value.(int) > target.(int)
		case float64:
			return value.(float64) > target.(float64)
		case int64:
			return value.(int64) > target.(int64)
		default:
			log.Printf("Can't compare %v and %v\n", value, target)
			return false
		}
	case ">=":
		switch value.(type) {
		case int:
			return value.(int) >= target.(int)
		case float64:
			return value.(float64) >= target.(float64)
		case int64:
			return value.(int64) >= target.(int64)
		default:
			log.Printf("Can't compare %v and %v\n", value, target)
			return false
		}

	case "<":
		switch value.(type) {
		case int:
			return value.(int) < target.(int)
		case float64:
			return value.(float64) < target.(float64)
		case int64:
			return value.(int64) < target.(int64)
		default:
			log.Printf("Can't compare %v and %v\n", value, target)
			return false
		}

	case "<=":
		switch value.(type) {
		case int:
			return value.(int) <= target.(int)
		case float64:
			return value.(float64) <= target.(float64)
		case int64:
			return value.(int64) <= target.(int64)
		default:
			log.Printf("Can't compare %v and %v\n", value, target)
			return false
		}
	}
	return false
}

type Receiver struct {
	Device DeviceInterface
	Key    string
	Value  string
}

func (t *Trigger) String() string {
	return fmt.Sprintf("%s.%s %s %v", t.Device.GetName(), t.Key, t.Condition.Operator, t.Condition.Value)
}

func (r *Receiver) String() string {
	return fmt.Sprintf("Receiver %s.%s = %s", r.Device.GetName(), r.Key, r.Value)
}

func (r *Rule) String() string {
	s := fmt.Sprintf("Rule with %d triggers and %d receivers:\n", len(r.Triggers), len(r.Receivers))
	for _, trigger := range r.Triggers {
		s += fmt.Sprintf("\t%s \n", trigger.String())
	}
	for _, receiver := range r.Receivers {
		s += fmt.Sprintf("\t=> %s\n", receiver.String())
	}
	return s
}

func NewRule(config core.RuleConfig, devices map[string]DeviceInterface) *Rule {
	r := Rule{}
	for _, triggerConfig := range config.Triggers {
		if _, ok := devices[triggerConfig.DeviceName]; !ok {
			log.Printf("Device %s not found\n", triggerConfig.DeviceName)
			continue
		}
		trigger := Trigger{
			Device: devices[triggerConfig.DeviceName],
			Key:    triggerConfig.Key,
			Condition: core.ToPtr(condition{
				Operator: triggerConfig.Condition.Operator,
				Value:    triggerConfig.Condition.Value,
			}),
		}
		r.Triggers = append(r.Triggers, trigger)
		devices[triggerConfig.DeviceName].AddRule(&r)
	}

	for _, receiverConfig := range config.Receivers {
		if _, ok := devices[receiverConfig.DeviceName]; !ok {
			log.Printf("Device %s not found\n", receiverConfig.DeviceName)
			continue
		}
		receiver := Receiver{
			Device: devices[receiverConfig.DeviceName],
			Key:    receiverConfig.Key,
			Value:  receiverConfig.Value,
		}
		r.Receivers = append(r.Receivers, receiver)
	}

	log.Printf("Created rule %s\n", r.String())
	return &r
}

func (r *Rule) Fire(channel chan core.SwitchRequest) []Receiver {
	log.Printf("[%32s] Firing %v\n", "Rules", r)
	requests := make(map[string]core.SwitchRequest)
	var firedReceivers []Receiver
	for _, receiver := range r.Receivers {
		request, ok := requests[receiver.Device.GetName()]
		if !ok {
			request = core.SwitchRequest{Device: receiver.Device.GetName()}
		}
		request.Command = receiver.Key
		switch receiver.Key {
		case "brightness":
			request.Value = receiver.Value
		case "duration":
			duration, err := strconv.Atoi(receiver.Value)
			if err != nil {
				log.Printf("Error parsing duration %s: %s\n", receiver.Value, err)
				continue
			}
			request.Duration = duration
		}
		requests[receiver.Device.GetName()] = request
		firedReceivers = append(firedReceivers, receiver)
	}

	for _, request := range requests {
		channel <- request
	}

	return firedReceivers
}

func makeComparable(value any, target any) (any, any, error) {
	if reflect.TypeOf(value) == reflect.TypeOf(target) {
		return value, target, nil
	}
	if value == nil || target == nil {
		return value, target, nil
	}
	switch value.(type) {
	case int:
		switch target.(type) {
		case int:
			return value, target, nil
		case float64:
			return float64(value.(int)), target, nil
		}
	case int64:
		switch target.(type) {
		case int:
			return int(value.(int64)), target, nil
		case float64:
			return float64(value.(int64)), target, nil
		}
	case float64:
		switch target.(type) {
		case int:
			return value, float64(target.(int)), nil
		case float64:
			return value, target, nil
		}
	case string:
		switch target.(type) {
		case string:
			return value, target, nil
		case int:
			return value, fmt.Sprintf("%d", target), nil
		case float64:
			return value, fmt.Sprintf("%f", target), nil
		}
	}
	return nil, nil, fmt.Errorf("can't compare %v and %v", value, target)
}

func (r *Rule) CheckTriggers() bool {
	matches := 0
	for _, trigger := range r.Triggers {
		if trigger.Condition.check() {
			matches++
		}
	}

	return matches > 0 && matches == len(r.Triggers)
}

func (r *Rule) ClearTriggers() {
	for _, trigger := range r.Triggers {
		trigger.Condition.Clear()
	}
}
