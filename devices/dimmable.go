package devices

import (
	core "github.com/PhilGruber/dimmy/core"
	"log"
	"math"
	"strconv"
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
}

func (d *Dimmable) GetMin() int {
	return d.Min
}

func (d *Dimmable) GetMax() int {
	return d.Max
}

func (d *Dimmable) GetStep() float64 {
	return d.Step
}

func (d *Dimmable) setStep(step float64) {
	d.Step = step
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
	return d.Target
}

func (d *Dimmable) setTarget(target float64) {
	d.Target = target
}

func (d *Dimmable) ProcessRequest(request core.SwitchRequest) {
	value, _ := strconv.ParseFloat(request.Value, 64)
	value = math.Min(value, 100)
	value = math.Max(value, 0)

	d.setTarget(value)

	log.Printf("Setting %s to %f within %d seconds\n", d.GetMqttTopic(), value, request.Duration)

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
	if current != d.Target {
		if d.transition {
			d.SetCurrent(d.Target)
			return d.Target, true
		}
		if d.Step == 0 {
			d.setStep(100)
		}
		if current > d.Target {
			current -= d.Step
			current = math.Max(current, d.Target)
		} else {
			current += d.Step
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
