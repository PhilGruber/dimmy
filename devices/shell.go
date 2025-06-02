package devices

import (
	"fmt"
	"github.com/PhilGruber/dimmy/core"
	"os/exec"
)

type Shell struct {
	Device
}

func NewShell(config core.DeviceConfig) *Shell {
	s := Shell{}
	s.Emoji = "#️"
	s.setBaseConfig(config)
	s.Hidden = true

	s.Receivers = []string{"command"}

	return &s
}

func (s *Shell) ProcessRequest(request core.SwitchRequest) {
	if request.Command == "command" {
		go s.execCommand(request.Value)
	}
}

func (s *Shell) execCommand(command string) {
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("error executing command '%s': %v, output: %s", command, err, output)
	}
}

func (s *Shell) UpdateValue() (float64, bool) {
	return 0, false // Shell does not have a value to update
}
