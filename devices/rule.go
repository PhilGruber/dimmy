package devices

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/PhilGruber/dimmy/core"
)

type Rule struct {
	Triggers  []Trigger
	Receivers []Receiver
	SingleUse bool
}

type Trigger struct {
	Device    DeviceInterface
	Key       string
	Condition *condition
}

type condition struct {
	Operator    string
	Value       any
	Delay       *int
	LastValue   any
	LastChanged *time.Time
}

func (c *condition) Clear() {
	c.LastValue = nil
	c.LastChanged = nil
}

func (c *condition) check() bool {
	needsNumeric := c.Operator == ">" || c.Operator == ">=" || c.Operator == "<" || c.Operator == "<="
	value, target, err := makeComparable(c.LastValue, c.Value, needsNumeric)
	if err != nil {
		log.Println(err)
		return false
	}

	if c.Delay != nil {
		if c.LastChanged == nil {
			return false
		}
		if time.Since(*c.LastChanged) < time.Duration(*c.Delay)*time.Second {
			return false
		}
	}

	//	fmt.Printf("\t --> Checking condition %p: %v %s %v\n", c, value, c.Operator, target)
	if value == nil || target == nil {
		return false
	}

	switch c.Operator {
	case "==":
		return value == target
	case "=":
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
			log.Printf("Can't compare %v [%T] > %v [%T]\n", value, value, target, target)
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
			log.Printf("Can't compare %v >= %v\n", value, target)
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
			log.Printf("Can't compare %v < %v\n", value, target)
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
			log.Printf("Can't compare %v <= %v\n", value, target)
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
	r.SingleUse = false
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
				Delay:    triggerConfig.Condition.Delay,
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

	log.Printf("Created %s\n", r.String())
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
		request.Key = receiver.Key
		switch receiver.Key {
		case "duration":
			duration, err := strconv.Atoi(receiver.Value)
			if err != nil {
				log.Printf("Error parsing duration %s: %s\n", receiver.Value, err)
				continue
			}
			request.Duration = duration
		default:
			request.Value = receiver.Value
		}
		requests[receiver.Device.GetName()] = request
		firedReceivers = append(firedReceivers, receiver)
	}

	for _, request := range requests {
		channel <- request
	}

	return firedReceivers
}

func makeComparable(value any, target any, numeric bool) (any, any, error) {
	if reflect.TypeOf(value) == reflect.TypeOf(target) {
		return value, target, nil
	}
	if value == nil || target == nil {
		return value, target, nil
	}
	if numeric && reflect.TypeOf(value).Kind() == reflect.String {
		var err error
		value, err = strconv.ParseFloat(value.(string), 64)
		if err != nil {
			return nil, nil, fmt.Errorf("can't convert %v [%s] to number: %s", value, reflect.TypeOf(value), err.Error())
		}
	}
	switch value.(type) {
	case int:
		switch target.(type) {
		case int:
			return value, target, nil
		case float64:
			return float64(value.(int)), target, nil
		case string:
			targetInt, err := strconv.Atoi(target.(string))
			if err != nil {
				return fmt.Sprintf("%d", value), target, nil
			}
			return value, targetInt, nil
		}
	case int64:
		switch target.(type) {
		case int:
			return int(value.(int64)), target, nil
		case float64:
			return float64(value.(int64)), target, nil
		case string:
			targetInt, err := strconv.Atoi(target.(string))
			if err != nil {
				return fmt.Sprintf("%d", value), target, nil
			}
			return value, int64(targetInt), nil
		}
	case float64:
		switch target.(type) {
		case int:
			return value, float64(target.(int)), nil
		case float64:
			return value, target, nil
		case string:
			targetFloat, err := strconv.ParseFloat(target.(string), 64)
			if err != nil {
				return fmt.Sprintf("%d", value), target, nil
			}
			return value, targetFloat, nil
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
	case bool:
		switch target.(type) {
		case bool:
			return value, target, nil
		case string:
			if strings.ToLower(target.(string)) == "true" {
				return value, true, nil
			}
			if strings.ToLower(target.(string)) == "false" {
				return value, false, nil
			}
		}
	}
	return nil, nil, fmt.Errorf("can't compare %v [%s] and %v [%s]", value, reflect.TypeOf(value), target, reflect.TypeOf(target))
}

func (r *Rule) CheckTriggers() bool {
	matches := 0
	for _, trigger := range r.Triggers {
		if !trigger.Condition.check() {
			return false
		}
		matches++
	}

	return matches > 0 && matches == len(r.Triggers)
}

func (t *Trigger) IsPersistent() bool {
	return t.Device.IsPersistent(t.Key)
}

func (r *Rule) ClearTriggers() {
	for _, trigger := range r.Triggers {
		if !trigger.IsPersistent() {
			trigger.Condition.Clear()
		}
	}
}
