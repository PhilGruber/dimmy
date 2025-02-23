package devices

import (
	"fmt"
	"github.com/PhilGruber/dimmy/core"
	"log"
	"reflect"
	"strconv"
)

type Rule struct {
	Triggers  []Trigger
	Receivers []Receiver
}

type Trigger struct {
	Device    DeviceInterface
	Key       string
	Condition struct {
		Operator string
		Value    any
	}
}

type Receiver struct {
	Device DeviceInterface
	Key    string
	Value  string
}

func NewRule(config core.RuleConfig, devices map[string]DeviceInterface) *Rule {
	r := Rule{}
	log.Printf("Creating rule %v\n", config)
	for _, triggerConfig := range config.Triggers {
		log.Println("Creating trigger")
		if _, ok := devices[triggerConfig.DeviceName]; !ok {
			log.Printf("Device %s not found\n", triggerConfig.DeviceName)
			continue
		}
		trigger := Trigger{
			Device: devices[triggerConfig.DeviceName],
			Key:    triggerConfig.Key,
			Condition: struct {
				Operator string
				Value    any
			}{
				Operator: triggerConfig.Condition.Operator,
				Value:    triggerConfig.Condition.Value,
			},
		}
		r.Triggers = append(r.Triggers, trigger)
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

	return &r
}

func (r *Rule) Fire(channel chan core.SwitchRequest) []Receiver {
	log.Printf("Firing rule %v\n", r)
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
	}
	return nil, nil, fmt.Errorf("can't compare %v and %v", value, target)
}

func (r *Rule) checkCondition(value any, condition string, target any) bool {
	//	log.Printf("Checking condition %v %s %v\n", value, condition, target)
	value, target, err := makeComparable(value, target)
	if err != nil {
		log.Println(err)
		return false
	}
	if value == nil || target == nil {
		return false
	}

	switch condition {
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
	}
	return false
}

func (r *Rule) CheckTrigger(device DeviceInterface, key string, value any) bool {
	for _, trigger := range r.Triggers {
		if trigger.Device.GetName() == device.GetName() && trigger.Key == key {
			return r.checkCondition(value, trigger.Condition.Operator, trigger.Condition.Value)
		}
	}
	return false
}

func (r *Rule) CheckTriggers() bool {
	matches := 0
	for _, trigger := range r.Triggers {
		if r.checkCondition(
			trigger.Device.GetTriggerValue(trigger.Key),
			trigger.Condition.Operator,
			trigger.Condition.Value) {
			matches++
		}
	}

	return matches > 0 && matches == len(r.Triggers)
}

func (r *Rule) ClearTriggers() {
	for _, trigger := range r.Triggers {
		trigger.Device.ClearTrigger(trigger.Key)
	}
}
