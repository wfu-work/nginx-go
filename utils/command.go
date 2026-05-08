package utils

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

type CommandResult struct {
	Command    string `json:"command"`
	Output     string `json:"output"`
	Success    bool   `json:"success"`
	DurationMs int64  `json:"durationMs"`
}

func RunCommand(timeout time.Duration, name string, args ...string) CommandResult {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	result := CommandResult{
		Command:    strings.TrimSpace(name + " " + strings.Join(args, " ")),
		Output:     strings.TrimSpace(string(output)),
		Success:    err == nil,
		DurationMs: time.Since(start).Milliseconds(),
	}
	if ctx.Err() == context.DeadlineExceeded {
		result.Output = strings.TrimSpace(result.Output + "\ncommand timeout")
	}
	if err != nil && result.Output == "" {
		result.Output = err.Error()
	}
	return result
}
