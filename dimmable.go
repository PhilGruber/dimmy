package main

import (
)


type Dimmable struct {
    Device
    Target int `json:"target"`
    Step float64 `json:"-"`
    Min int `json:"-"`
    Max int `json:"-"`
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

func (d Dimmable) getTarget() int {
    return d.Target
}

func (d *Dimmable) setTarget(target int) {
    d.Target = target
}

func (d *Dimmable) UpdateValue() bool {
    if d.Current != float64(d.Target) {
        if (d.Current > float64(d.Target)) {
            d.Current -= d.Step
            if (d.Current <= float64(d.Target)) {
                d.Current = float64(d.Target)
            }
        } else {
            d.Current += d.Step
            if (d.Current >= float64(d.Target)) {
                d.Current = float64(d.Target)
            }
        }
        return true
    }
    return false
}
