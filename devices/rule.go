package devices

import (
	"github.com/PhilGruber/dimmy/core"
	"log"
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
		log.Println("Creating receiver")
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

func (r *Rule) checkCondition(value any, condition string, target any) bool {
	switch condition {
	case "==":
		return value == target
	case "!=":
		return value != target
	case ">":
		switch target.(type) {
		case int:
			return value.(int) > target.(int)
		case float64:
			return value.(float64) > target.(float64)
		}
	case "<":
		switch value.(type) {
		case int:
			return value.(int) < target.(int)
		case float64:
			return value.(float64) < target.(float64)
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
