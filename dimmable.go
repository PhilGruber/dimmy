package main

import (
	"log"
	"math"
	"time"
)

type Dimmable struct {
	Device
	Target   int     `json:"target"`
	Step     float64 `json:"-"`
	Min      int     `json:"-"`
	Max      int     `json:"-"`
	LastSent int     `json:"-"`
}

func (d Dimmable) getMin() int {
	return d.Min
}

func (d Dimmable) getMax() int {
	return d.Max
}

func (d Dimmable) getStep() float64 {
	return d.Step
}

func (d *Dimmable) setStep(step float64) {
	d.Step = step
}

func (d Dimmable) getLastSent() int {
	return d.LastSent
}

func (d *Dimmable) setLastSent(lastSent int) {
	d.LastSent = lastSent
}

func (d Dimmable) getLastChanged() *time.Time {
	return d.LastChanged
}

func (d *Dimmable) setLastChanged(lastSent *time.Time) {
	d.LastChanged = lastSent
}

func (d Dimmable) getTarget() int {
	return d.Target
}

func (d *Dimmable) setTarget(target int) {
	d.Target = target
}

func (d *Dimmable) processRequest(request SwitchRequest) {
	request.Value = int(math.Min(float64(request.Value), float64(d.getMax())))
	request.Value = int(math.Max(float64(request.Value), float64(d.getMin())))

	d.setTarget(request.Value)
	diff := int(math.Abs(d.getCurrent() - float64(request.Value)))
	var step float64
	cycles := request.Duration * 1000 / cycleLength
	if request.Duration == 0 {
		step = float64(diff)
	} else {
		step = float64(diff) / float64(cycles)
	}

	log.Printf("Dimming %s from %.f to %d: %d steps in %d seconds (%.1f steps per cycle)", request.Device, d.getCurrent(), request.Value, diff, request.Duration, step)
	d.setStep(step)

}

func (d *Dimmable) UpdateValue() (float64, bool) {
	current := d.getCurrent()
	if current != float64(d.Target) {
		if current > float64(d.Target) {
			current -= d.Step
			if current <= float64(d.Target) {
				current = float64(d.Target)
			}
		} else {
			current += d.Step
			if current >= float64(d.Target) {
				current = float64(d.Target)
			}
		}
		//		log.Printf("%s.setCurrent(%f)\n", d.Type, current)
		d.setCurrent(current)
		return current, true
	}
	return 0, false
}
