package worker

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type Launcher struct {
	Command    string
	SocketPath string
}

type Spec struct {
	AgentID string
	Role    string
	TaskID  string
}

func (l Launcher) Start(ctx context.Context, spec Spec) (*exec.Cmd, error) {
	parts := strings.Fields(l.Command)
	if len(parts) == 0 {
		return nil, fmt.Errorf("worker command is empty")
	}
	args := append([]string{}, parts[1:]...)
	args = append(args,
		"--agent-id", spec.AgentID,
		"--role", spec.Role,
		"--task-id", spec.TaskID,
		"--sock", l.SocketPath,
		"--work-ms", strconv.Itoa(800),
	)
	cmd := exec.CommandContext(ctx, parts[0], args...)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}
