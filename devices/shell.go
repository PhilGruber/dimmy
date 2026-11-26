package devices

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/PhilGruber/dimmy/core"
)

type Shell struct {
	Device

	commands core.Commands
}

func NewShell(config core.DeviceConfig) *Shell {
	s := Shell{}
	s.Icon = "#️"
	s.setBaseConfig(config)
	s.Hidden = true
	s.Type = "shell"

	if config.Options != nil && config.Options.Commands != nil {
		s.commands = *config.Options.Commands
	}

	s.Receivers = []string{"command"}

	return &s
}

func (s *Shell) ProcessRequest(request core.SwitchRequest) {
	if request.Key == "command" {
		go s.execCommand(request.Value)
		return
	}

	command, ok := s.commands.Get(request.Value)
	if !ok {
		log.Printf("Device %s does not support command %s. Please define this in config file.\n", s.Name, request.Value)
		return
	}

	go s.execCommand(command)

}

func (s *Shell) execCommand(command string) {
	args, err := ParseScriptCommand(command)
	if err != nil {
		log.Printf("Rejected script: %v", err)
		return
	}
	file, err := openScript(args[0])
	if err != nil {
		log.Printf("Rejected script: %v", err)
		return
	}
	defer file.Close()
	// Execute the validated open file, so a path replacement cannot redirect
	// execution outside the scripts directory between validation and exec.
	cmd := scriptCommand(file, args[1:])
	output, err := cmd.CombinedOutput()
	fmt.Printf("executed command '%s': %v, output: %s\n", command, err, output)
	if err != nil {
		fmt.Printf("error executing command '%s': %v, output: %s\n", command, err, output)
	}
}

func scriptCommand(file *os.File, args []string) *exec.Cmd {
	cmd := exec.Command("/proc/self/fd/3", args...)
	cmd.ExtraFiles = []*os.File{file}
	return cmd
}

// ParseScriptCommand supports quoted arguments without shell expansion or evaluation.
func ParseScriptCommand(command string) ([]string, error) {
	var args []string
	var word strings.Builder
	var quote rune
	escaped, started := false, false
	for _, char := range command {
		if char == 0 {
			return nil, fmt.Errorf("NUL is not allowed")
		}
		if escaped {
			word.WriteRune(char)
			escaped = false
			started = true
			continue
		}
		if char == '\\' && quote != '\'' {
			escaped = true
			started = true
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
			} else {
				word.WriteRune(char)
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			started = true
			continue
		}
		if char == ' ' || char == '\t' || char == '\n' || char == '\r' {
			if started {
				args = append(args, word.String())
				word.Reset()
				started = false
			}
			continue
		}
		word.WriteRune(char)
		started = true
	}
	if escaped || quote != 0 {
		return nil, fmt.Errorf("unterminated quoted argument")
	}
	if started {
		args = append(args, word.String())
	}
	if len(args) == 0 || args[0] == "" {
		return nil, fmt.Errorf("a script name is required")
	}
	return args, nil
}

func openScript(name string) (*os.File, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return openScriptInDirectory(filepath.Join(home, "scripts"), name)
}

func openScriptInDirectory(directory, name string) (*os.File, error) {
	if name == "." || name == ".." || filepath.Base(name) != name || strings.ContainsAny(name, "/\\") {
		return nil, fmt.Errorf("scripts must be filenames inside $HOME/scripts")
	}
	// Pin the directory without following a symlink, including during rename races.
	directoryFile, err := os.OpenFile(directory, os.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	defer directoryFile.Close()
	root, err := os.OpenRoot(fmt.Sprintf("/proc/self/fd/%d", directoryFile.Fd()))
	if err != nil {
		return nil, err
	}
	defer root.Close()
	file, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err == nil && (!info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0) {
		err = fmt.Errorf("script must be a regular executable file")
	}
	if err == nil {
		var header [2]byte
		_, err = file.ReadAt(header[:], 0)
		if err == nil && string(header[:]) != "#!" {
			err = fmt.Errorf("script must have a shebang")
		}
	}
	if err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

type ShellScriptInventory struct {
	Directory string
	Scripts   []string
	Rejected  map[string]string
	Error     string
}

func (s *Shell) ScriptInventory() ShellScriptInventory {
	home, err := os.UserHomeDir()
	if err != nil {
		return ShellScriptInventory{Error: err.Error()}
	}
	return scriptInventory(filepath.Join(home, "scripts"))
}

func scriptInventory(directory string) ShellScriptInventory {
	inventory := ShellScriptInventory{Directory: directory, Scripts: []string{}, Rejected: make(map[string]string)}
	entries, err := os.ReadDir(directory)
	if err != nil {
		inventory.Error = err.Error()
		return inventory
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		file, err := openScriptInDirectory(directory, entry.Name())
		if err != nil {
			inventory.Rejected[entry.Name()] = err.Error()
			continue
		}
		file.Close()
		inventory.Scripts = append(inventory.Scripts, entry.Name())
	}
	return inventory
}

func (s *Shell) Scripts() []string {
	return s.ScriptInventory().Scripts
}

func (s *Shell) ValidateCommand(command string) error {
	args, err := ParseScriptCommand(command)
	if err != nil {
		return err
	}
	file, err := openScript(args[0])
	if err != nil {
		return err
	}
	return file.Close()
}

func (s *Shell) GetCommands() []string {
	return s.commands.Names()
}

func (s *Shell) UpdateValue() (float64, bool) {
	return 0, false // Shell does not have a value to update
}

func (s *Shell) IsPseudoDevice() bool {
	return len(s.commands) == 0
}
