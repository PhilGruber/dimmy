package devices

import (
	"testing"

	"github.com/PhilGruber/dimmy/core"
)

// newTestLight returns a properly initialised Light (and thus Dimmable) for
// testing Dimmable behaviour without needing a real MQTT connection.
func newTestLight() *Light {
	return NewLight(core.DeviceConfig{
		Name:  "test-light",
		Topic: "cmnd/test/dimmer",
	})
}

func TestDimmable_ProcessRequest_AbsoluteSetsTarget(t *testing.T) {
	d := newTestLight()
	d.ProcessRequest(core.SwitchRequest{Value: "75"})
	if d.GetTarget() != 75 {
		t.Errorf("expected Target=75, got %f", d.GetTarget())
	}
}

func TestDimmable_ProcessRequest_RelativeAdds(t *testing.T) {
	d := newTestLight()
	d.SetCurrent(30)
	d.ProcessRequest(core.SwitchRequest{Value: "+20"})
	if d.GetTarget() != 50 {
		t.Errorf("expected Target=50, got %f", d.GetTarget())
	}
}

func TestDimmable_ProcessRequest_RelativeSubtracts(t *testing.T) {
	d := newTestLight()
	d.SetCurrent(30)
	d.ProcessRequest(core.SwitchRequest{Value: "-10"})
	if d.GetTarget() != 20 {
		t.Errorf("expected Target=20, got %f", d.GetTarget())
	}
}

func TestDimmable_ProcessRequest_ClampsAtMax(t *testing.T) {
	d := newTestLight()
	d.ProcessRequest(core.SwitchRequest{Value: "150"})
	if d.GetTarget() != 100 {
		t.Errorf("expected Target clamped to 100, got %f", d.GetTarget())
	}
}

func TestDimmable_ProcessRequest_ClampsAtMin(t *testing.T) {
	d := newTestLight()
	d.SetCurrent(50)
	d.ProcessRequest(core.SwitchRequest{Value: "-100"})
	if d.GetTarget() != 0 {
		t.Errorf("expected Target clamped to 0, got %f", d.GetTarget())
	}
}

func TestDimmable_ProcessRequest_StepInstantWhenDurationZero(t *testing.T) {
	d := newTestLight()
	// current=0, value=60, Duration=0 → step = diff = 60
	d.ProcessRequest(core.SwitchRequest{Value: "60"})
	if d.GetStep() != 60 {
		t.Errorf("expected Step=60, got %f", d.GetStep())
	}
}

func TestDimmable_ProcessRequest_StepWithDuration(t *testing.T) {
	d := newTestLight()
	// current=0, value=75, Duration=5s
	// diff=75, cycles=5000/200=25, step=3.0
	d.ProcessRequest(core.SwitchRequest{Value: "75", Duration: 5})
	if d.GetStep() != 3.0 {
		t.Errorf("expected Step=3.0, got %f", d.GetStep())
	}
}

func TestDimmable_UpdateValue_AlreadyAtTarget(t *testing.T) {
	d := newTestLight()
	// Current=0, Target=0 → no change needed
	val, send := d.UpdateValue()
	if send {
		t.Error("expected send=false when already at target")
	}
	if val != 0 {
		t.Errorf("expected val=0, got %f", val)
	}
}

func TestDimmable_UpdateValue_StepsUp(t *testing.T) {
	d := newTestLight()
	// step=3, current=0 → after one call: current=3
	d.ProcessRequest(core.SwitchRequest{Value: "75", Duration: 5})
	val, send := d.UpdateValue()
	if !send {
		t.Error("expected send=true when stepping toward target")
	}
	if val != 3.0 {
		t.Errorf("expected val=3.0 after one step, got %f", val)
	}
}

func TestDimmable_UpdateValue_StepsDown(t *testing.T) {
	d := newTestLight()
	d.SetCurrent(60)
	// diff=30, cycles=25, step=1.2
	d.ProcessRequest(core.SwitchRequest{Value: "30", Duration: 5})
	val, send := d.UpdateValue()
	if !send {
		t.Error("expected send=true when stepping down")
	}
	if val >= 60 || val <= 30 {
		t.Errorf("expected 30 < val < 60, got %f", val)
	}
}

func TestDimmable_UpdateValue_DoesNotOvershoot(t *testing.T) {
	d := newTestLight()
	// step=50, target=50 → single call lands exactly on target
	d.ProcessRequest(core.SwitchRequest{Value: "50"})
	val, send := d.UpdateValue()
	if !send {
		t.Error("expected send=true on first call")
	}
	if val != 50 {
		t.Errorf("expected val=50 (clamped at target), got %f", val)
	}
	_, send2 := d.UpdateValue()
	if send2 {
		t.Error("expected send=false once target is reached")
	}
}

func TestDimmable_UpdateValue_TransitionJumpsImmediately(t *testing.T) {
	d := newTestLight()
	d.transition = true
	d.setTarget(80)
	val, send := d.UpdateValue()
	if !send {
		t.Error("expected send=true in transition mode")
	}
	if val != 80 {
		t.Errorf("expected val=80, got %f", val)
	}
	if d.GetCurrent() != 80 {
		t.Errorf("expected Current=80 after transition jump, got %f", d.GetCurrent())
	}
}
