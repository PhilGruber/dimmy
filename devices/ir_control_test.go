package devices

import (
	"fmt"
	"testing"

	"github.com/PhilGruber/dimmy/core"
	"gopkg.in/yaml.v3"
)

func irControlConfig(t *testing.T, preventResending bool) core.DeviceConfig {
	t.Helper()
	raw := fmt.Sprintf(`name: test-ir
topic: ir/device
options:
  commands:
    power: IR_POWER
    mute: IR_MUTE
  prevent_resending: %v`, preventResending)
	var cfg core.DeviceConfig
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}
	return cfg
}

func TestIRControl_ProcessRequest_KnownCommand(t *testing.T) {
	ir := NewIrControl(irControlConfig(t, false))
	ir.ProcessRequest(core.SwitchRequest{Value: "power"})
	if ir.nextRequest == nil {
		t.Fatal("expected nextRequest to be set for known command")
	}
	if ir.nextRequest.IrCode != "IR_POWER" {
		t.Errorf("expected IrCode=IR_POWER, got %s", ir.nextRequest.IrCode)
	}
}

func TestIRControl_ProcessRequest_UnknownCommand(t *testing.T) {
	ir := NewIrControl(irControlConfig(t, false))
	ir.ProcessRequest(core.SwitchRequest{Value: "unknown"})
	if ir.nextRequest != nil {
		t.Error("expected nextRequest to remain nil for unknown command")
	}
}

func TestIRControl_ProcessRequest_PreventResendingBlocksDuplicate(t *testing.T) {
	ir := NewIrControl(irControlConfig(t, true))
	ir.ProcessRequest(core.SwitchRequest{Value: "power"})
	// Simulate the command having been sent already.
	ir.lastCommand = ir.nextRequest.IrCode
	ir.nextRequest = nil
	// Same command again without Force — should be skipped.
	ir.ProcessRequest(core.SwitchRequest{Value: "power"})
	if ir.nextRequest != nil {
		t.Error("expected nextRequest to remain nil when preventResending blocks duplicate")
	}
}

func TestIRControl_ProcessRequest_ForceOverridesPreventResending(t *testing.T) {
	ir := NewIrControl(irControlConfig(t, true))
	ir.ProcessRequest(core.SwitchRequest{Value: "power"})
	ir.lastCommand = ir.nextRequest.IrCode
	ir.nextRequest = nil
	// Force=true must bypass the duplicate guard.
	ir.ProcessRequest(core.SwitchRequest{Value: "power", Force: true})
	if ir.nextRequest == nil {
		t.Error("expected nextRequest to be set when Force=true overrides preventResending")
	}
}

func TestIRControl_UpdateValue_NoPendingRequest(t *testing.T) {
	ir := NewIrControl(irControlConfig(t, false))
	_, send := ir.UpdateValue()
	if send {
		t.Error("expected send=false when no pending request")
	}
}

func TestIRControl_UpdateValue_PendingRequest(t *testing.T) {
	ir := NewIrControl(irControlConfig(t, false))
	ir.ProcessRequest(core.SwitchRequest{Value: "power"})
	_, send := ir.UpdateValue()
	if !send {
		t.Error("expected send=true when nextRequest is set")
	}
}

func TestIRControl_SetReceiverValue_Command(t *testing.T) {
	ir := NewIrControl(irControlConfig(t, false))
	ir.SetReceiverValue("command", "mute")
	if ir.nextRequest == nil {
		t.Fatal("expected nextRequest to be set after SetReceiverValue")
	}
	if ir.nextRequest.IrCode != "IR_MUTE" {
		t.Errorf("expected IrCode=IR_MUTE, got %s", ir.nextRequest.IrCode)
	}
}

func TestIRControl_GetCommands_ReturnsAllKeys(t *testing.T) {
	ir := NewIrControl(irControlConfig(t, false))
	cmds := ir.GetCommands()
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}
	found := map[string]bool{}
	for _, c := range cmds {
		found[c] = true
	}
	if !found["power"] {
		t.Error("expected 'power' in commands")
	}
	if !found["mute"] {
		t.Error("expected 'mute' in commands")
	}
}
