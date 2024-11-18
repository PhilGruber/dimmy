package devices

import (
	"github.com/PhilGruber/dimmy/core"
	"log"
)

type Rule struct {
	Triggers []struct {
		device    DeviceInterface
		key       string
		condition struct {
			Operator string
			Value    any
		}
	}
	Receivers []struct {
		device DeviceInterface
		key    string
		value  any
	}
}

func NewRule(config core.RuleConfig, devices map[string]DeviceInterface) *Rule {
	r := Rule{}
	log.Printf("Creating rule %v\n", config)
	for _, triggerConfig := range config.Triggers {
		log.Println("Creating trigger")
		trigger := struct {
			device    DeviceInterface
			key       string
			condition struct {
				Operator string
				Value    any
			}
		}{
			device: devices[triggerConfig.DeviceName],
			key:    triggerConfig.Key,
			condition: struct {
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
		receiver := struct {
			device DeviceInterface
			key    string
			value  any
		}{
			device: devices[receiverConfig.DeviceName],
			key:    receiverConfig.Key,
			value:  receiverConfig.Value,
		}
		r.Receivers = append(r.Receivers, receiver)
	}

	return &r
}

func (r *Rule) Fire(channel chan core.SwitchRequest) {
	log.Printf("Firing rule %v\n", r)
	var requests map[string]core.SwitchRequest
	for _, receiver := range r.Receivers {
		request, ok := requests[receiver.device.GetName()]
		if !ok {
			request = core.SwitchRequest{Device: receiver.device.GetName()}
		}
		switch receiver.key {
		case "value":
			request.Value = receiver.value.(float64)
		case "duration":
			request.Duration = receiver.value.(int)
		}
		requests[receiver.device.GetName()] = request
	}

	for _, request := range requests {
		channel <- request
	}
}

func (r *Rule) CheckTrigger(device DeviceInterface, key string, value any) bool {
	for _, trigger := range r.Triggers {
		if trigger.device.GetName() == device.GetName() && trigger.key == key {
			return r.checkCondition(value, trigger.condition.Operator, trigger.condition.Value)
		}
	}
	return false
}

func (r *Rule) checkCondition(value any, condition string, target any) bool {
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

func (r *Rule) CheckTriggers() bool {
	matches := 0
	for _, trigger := range r.Triggers {
		if r.checkCondition(trigger.device.GetTriggerValue(trigger.key), trigger.condition.Operator, trigger.condition.Value) {
			matches++
		}
	}

	return matches > 0 && matches == len(r.Triggers)
}