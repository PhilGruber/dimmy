package devices

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/PhilGruber/dimmy/core"
)

type Shell struct {
	Device

	commands map[string]string
}

func NewShell(config core.DeviceConfig) *Shell {
	s := Shell{}
	s.Icon = "#️"
	s.setBaseConfig(config)
	s.Hidden = true
	s.Type = "shell"

	if config.Options != nil {
		s.commands = *config.Options.Commands
	}

	s.Receivers = []string{"command"}

	return &s
}

func (s *Shell) ProcessRequest(request core.SwitchRequest) {
	if request.Key == "command" {
		// TODO: sanitize this
		go s.execCommand(request.Value)
		return
	}

	command, ok := s.commands[request.Value]
	if !ok {
		log.Printf("Device %s does not support command %s. Please define this in config file.\n", s.Name, request.Value)
		return
	}

	go s.execCommand(command)

}

func (s *Shell) execCommand(command string) {
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	fmt.Printf("executed command '%s': %v, output: %s", command, err, output)
	if err != nil {
		fmt.Printf("error executing command '%s': %v, output: %s", command, err, output)
	}
}

func (s *Shell) GetCommands() []string {
	var commands []string
	for k := range s.commands {
		commands = append(commands, k)
	}
	return commands
}

func (s *Shell) UpdateValue() (float64, bool) {
	return 0, false // Shell does not have a value to update
}
