package copilot

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

const endpointPath = "/copilot_internal/user"

type Client interface {
	Current(context.Context) (UserData, error)
}

type CommandRunner interface {
	Run(context.Context, string, ...string) ([]byte, error)
}

type ExecCommandRunner struct{}

func (ExecCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

type GHClient struct {
	runner  CommandRunner
	command string
	timeout time.Duration
}

func NewGHClient(runner CommandRunner, timeout time.Duration) GHClient {
	return NewGHClientWithCommand(runner, timeout, "gh")
}

func NewGHClientWithCommand(runner CommandRunner, timeout time.Duration, command string) GHClient {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	if command == "" {
		command = "gh"
	}
	return GHClient{runner: runner, command: command, timeout: timeout}
}

func ResolveGHCommand() string {
	if path, err := exec.LookPath("gh"); err == nil {
		return path
	}
	for _, path := range []string{
		"/opt/homebrew/bin/gh",
		"/usr/local/bin/gh",
		"/usr/bin/gh",
	} {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	return "gh"
}

func (client GHClient) Current(parent context.Context) (UserData, error) {
	ctx, cancel := context.WithTimeout(parent, client.timeout)
	defer cancel()

	data, err := client.runner.Run(
		ctx,
		client.command,
		"api",
		endpointPath,
		"--hostname", "github.com",
		"--header", "Accept: application/json",
		"--header", "User-Agent: gh-copilot-credits",
	)
	if err != nil {
		if ctx.Err() != nil {
			return UserData{}, fmt.Errorf("gh api request timed out")
		}
		return UserData{}, fmt.Errorf("gh api request failed: %w", err)
	}
	return DecodeUserData(bytes.NewReader(data))
}
