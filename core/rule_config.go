package core

import (
	"errors"
	"time"
)

func (r RuleConfig) Validate() error {
	if err := r.ValidateTriggerFlags(); err != nil {
		return err
	}
	for _, receiver := range r.Receivers {
		if receiver.Delay < 0 || int64(receiver.Delay) > (1<<63-1)/int64(time.Second) {
			return errors.New("control delay must be a nonnegative number of seconds within the supported duration range")
		}
	}
	return nil
}

func (r RuleConfig) ValidateTriggerFlags() error {
	count := 0
	for _, trigger := range r.Triggers {
		if trigger.OnChange {
			count++
		}
	}
	if count > 1 {
		return errors.New("only one trigger per rule may have on_change enabled")
	}
	return nil
}
