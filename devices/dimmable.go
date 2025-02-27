package devices

import (
	"github.com/PhilGruber/dimmy/core"
	"log"
	"math"
	"strconv"
	"sync"
	"time"
)

type Dimmable struct {
	Device
	Target         float64 `json:"target"`
	Step           float64 `json:"-"`
	Min            int     `json:"-"`
	Max            int     `json:"-"`
	LastSent       int     `json:"-"`
	transition     bool
	TransitionTime int
	targetLock     sync.RWMutex
	stepLock       sync.RWMutex
}

func (d *Dimmable) GetMin() int {
	return d.Min
}

func (d *Dimmable) GetMax() int {
	return d.Max
}

func (d *Dimmable) GetStep() float64 {
	d.stepLock.RLock()
	defer d.stepLock.RUnlock()
	return d.Step
}

func (d *Dimmable) setStep(step float64) {
	d.stepLock.Lock()
	d.Step = step
	d.stepLock.Unlock()
}

func (d *Dimmable) GetLastSent() int {
	return d.LastSent
}

func (d *Dimmable) setLastSent(lastSent int) {
	d.LastSent = lastSent
}

func (d *Dimmable) GetLastChanged() *time.Time {
	return d.LastChanged
}

func (d *Dimmable) setLastChanged(lastSent *time.Time) {
	d.LastChanged = lastSent
}

func (d *Dimmable) GetTarget() float64 {
	d.targetLock.RLock()
	defer d.targetLock.RUnlock()
	return d.Target
}

func (d *Dimmable) setTarget(target float64) {
	d.targetLock.Lock()
	d.Target = target
	d.targetLock.Unlock()
}

func (d *Dimmable) ProcessRequest(request core.SwitchRequest) {
	relativeValue := false
	if request.Value[0] == '+' || request.Value[0] == '-' {
		relativeValue = true
	}
	value, _ := strconv.ParseFloat(request.Value, 64)

	if relativeValue {
		value += d.GetCurrent()
	}

	value = math.Min(value, 100)
	value = math.Max(value, 0)

	d.setTarget(value)

	log.Printf("[%32s] Dimming to %3.1f within %d seconds\n", d.GetName(), value, request.Duration)

	if d.transition {
		d.TransitionTime = request.Duration
		d.setStep(0)
		return
	}

	diff := int(math.Abs(d.GetCurrent() - value))
	var step float64
	cycles := request.Duration * 1000 / core.CycleLength
	if request.Duration == 0 {
		step = float64(diff)
	} else {
		step = float64(diff) / float64(cycles)
	}
	d.setStep(step)
}

func (d *Dimmable) ProcessRequestChild(request core.SwitchRequest) {
	d.ProcessRequest(request)
}

func (d *Dimmable) UpdateValue() (float64, bool) {
	current := d.GetCurrent()
	if current != d.GetTarget() {
		if d.transition {
			d.SetCurrent(d.Target)
			return d.Target, true
		}
		if d.GetStep() == 0 {
			d.setStep(100)
		}
		if current > d.GetTarget() {
			current -= d.GetStep()
			current = math.Max(current, d.Target)
		} else {
			current += d.GetStep()
			current = math.Min(current, d.Target)
		}
		d.SetCurrent(current)
		return current, true
	}
	return 0, false
}

func (d *Dimmable) UpdateValueChild() (float64, bool) {
	return d.UpdateValue()
}
