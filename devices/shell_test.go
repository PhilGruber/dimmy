package devices

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScriptConfinementAndLiteralParameters(t *testing.T) {
	root := t.TempDir()
	directory := filepath.Join(root, "scripts")
	require.NoError(t, os.Mkdir(directory, 0700))
	script := filepath.Join(directory, "echo args.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\"\n"), 0700))
	outside := filepath.Join(root, "outside.sh")
	require.NoError(t, os.WriteFile(outside, []byte("#!/bin/sh\nexit 99\n"), 0700))
	require.NoError(t, os.Symlink(outside, filepath.Join(directory, "escape.sh")))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "binary"), []byte("not a script"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "not-executable.sh"), []byte("#!/bin/sh\n"), 0600))
	for _, name := range []string{"../outside.sh", outside, "escape.sh", "binary", "not-executable.sh", ".", "..", "/bin/sh"} {
		_, err := openScriptInDirectory(directory, name)
		require.Error(t, err, name)
	}
	args, err := ParseScriptCommand(`"echo args.sh" 'hello world' '$(touch SHOULD_NOT_EXIST)' ';' ""`)
	require.NoError(t, err)
	file, err := openScriptInDirectory(directory, args[0])
	require.NoError(t, err)
	defer file.Close()
	// Replacing the validated path must not change which script executes.
	require.NoError(t, os.Rename(script, script+".old"))
	require.NoError(t, os.Symlink(outside, script))
	output, err := scriptCommand(file, args[1:]).CombinedOutput()
	require.NoError(t, err, string(output))
	require.Equal(t, "hello world\n$(touch SHOULD_NOT_EXIST)\n;\n\n", string(output))
	linkedDirectory := filepath.Join(root, "linked-scripts")
	require.NoError(t, os.Symlink(directory, linkedDirectory))
	_, err = openScriptInDirectory(linkedDirectory, "binary")
	require.Error(t, err)
}

func TestScriptArgumentsRejectMalformedInput(t *testing.T) {
	for _, command := range []string{"", "\x00", `script.sh 'unclosed`, "script.sh trailing\\"} {
		_, err := ParseScriptCommand(command)
		require.Error(t, err)
	}
}

func TestScriptInventoryIncludesExecutableShellScript(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "test.sh"), []byte("#!/bin/sh\necho test\n"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "no-shebang.sh"), []byte("echo test\n"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "not-executable.sh"), []byte("#!/bin/sh\necho test\n"), 0644))
	inventory := scriptInventory(directory)
	require.Empty(t, inventory.Error)
	require.Equal(t, directory, inventory.Directory)
	require.Equal(t, []string{"test.sh"}, inventory.Scripts)
	require.Contains(t, inventory.Rejected["no-shebang.sh"], "shebang")
	require.Contains(t, inventory.Rejected["not-executable.sh"], "executable")
	missing := scriptInventory(filepath.Join(directory, "missing"))
	require.NotEmpty(t, missing.Error)
}
