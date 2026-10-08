package launchd

import (
	"context"
	"fmt"
	"os/exec"
)

type CommandRunner interface {
	Run(context.Context, string, ...string) error
}

type ExecCommandRunner struct{}

func (ExecCommandRunner) Run(ctx context.Context, name string, args ...string) error {
	return exec.CommandContext(ctx, name, args...).Run()
}

func Bootstrap(ctx context.Context, runner CommandRunner, domain, plistPath string) error {
	if err := runner.Run(ctx, "launchctl", "bootstrap", domain, plistPath); err != nil {
		return fmt.Errorf("launchctl bootstrap failed: %w", err)
	}
	return nil
}

func Bootout(ctx context.Context, runner CommandRunner, target string) error {
	if err := runner.Run(ctx, "launchctl", "bootout", target); err != nil {
		return fmt.Errorf("launchctl bootout failed: %w", err)
	}
	return nil
}

func Print(ctx context.Context, runner CommandRunner, target string) error {
	if err := runner.Run(ctx, "launchctl", "print", target); err != nil {
		return fmt.Errorf("launchctl print failed: %w", err)
	}
	return nil
}
