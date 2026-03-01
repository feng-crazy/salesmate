package tools

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ExecTool implements a tool to execute shell commands
type ExecTool struct {
	workingDir            string
	timeout              time.Duration
	restrictToWorkspace bool
}

// NewExecTool creates a new execute command tool
func NewExecTool(workingDir string, timeout int, restrictToWorkspace bool) *ExecTool {
	return &ExecTool{
		workingDir:            workingDir,
		timeout:              time.Duration(timeout) * time.Second,
		restrictToWorkspace: restrictToWorkspace,
	}
}

// Name returns the name of the tool
func (t *ExecTool) Name() string {
	return "execute_command"
}

// Description returns the description of the tool
func (t *ExecTool) Description() string {
	return "Execute a shell command"
}

// Call executes the tool with the given arguments
func (t *ExecTool) Call(args map[string]interface{}) (string, error) {
	command, ok := args["command"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'command' argument")
	}

	// If restricting to workspace, check that the command is not trying to escape
	if t.restrictToWorkspace {
		cmdParts := strings.Fields(command)
		if len(cmdParts) > 0 {
			// Basic check to see if command tries to access paths outside of workspace
			if strings.Contains(command, "../") || strings.HasPrefix(command, "/") {
				return "", fmt.Errorf("command violates workspace restriction: %s", command)
			}
		}
	}

	// Split command and arguments properly
	cmdParts := strings.Fields(command)
	if len(cmdParts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	name := cmdParts[0]
	var cmdArgs []string
	if len(cmdParts) > 1 {
		cmdArgs = cmdParts[1:]
	}

	cmd := exec.Command(name, cmdArgs...)
	cmd.Dir = t.workingDir

	// Set a timeout
	done := make(chan error, 1)
	output := make(chan string, 1)

	go func() {
		out, err := cmd.CombinedOutput()
		output <- string(out)
		done <- err
	}()

	select {
	case <-time.After(t.timeout):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return "", fmt.Errorf("command timed out after %v", t.timeout)
	case err := <-done:
		out := <-output
		if err != nil {
			return fmt.Sprintf("Command failed: %s\nOutput: %s", err.Error(), out), nil
		}
		return fmt.Sprintf("Command executed successfully:\n%s", out), nil
	}
}