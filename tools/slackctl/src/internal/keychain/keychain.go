package keychain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"

	"github.com/Alechan/ai-resources/tools/slackctl/src/internal/auth"
)

const service = "slackctl"

type Runner interface {
	Run(context.Context, []byte, ...string) ([]byte, error)
}

type commandRunner struct{}

func (commandRunner) Run(ctx context.Context, input []byte, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "/usr/bin/security", args...)
	cmd.Stdin = bytes.NewReader(input)
	return cmd.CombinedOutput()
}

type Store struct {
	runner Runner
}

func New() *Store                   { return NewStore(commandRunner{}) }
func NewStore(runner Runner) *Store { return &Store{runner: runner} }

func (s *Store) Save(ctx context.Context, credential auth.Credential) error {
	data, err := json.Marshal(credential)
	if err != nil {
		return errors.New("could not serialize credentials")
	}
	input := append(data, '\n')
	_, err = s.runner.Run(ctx, input, "add-generic-password", "-s", service, "-a", credential.WorkspaceHost, "-U", "-w")
	clear(input)
	clear(data)
	if err != nil {
		return errors.New("could not save credentials in macOS Keychain")
	}
	return nil
}

func (s *Store) Load(ctx context.Context, host string) (auth.Credential, error) {
	data, err := s.runner.Run(ctx, nil, "find-generic-password", "-s", service, "-a", host, "-w")
	if err != nil {
		if isMissingItem(err) {
			return auth.Credential{}, fmt.Errorf("%w for workspace %s", auth.ErrNotFound, host)
		}
		return auth.Credential{}, errors.New("could not read credentials from macOS Keychain")
	}
	defer clear(data)
	var credential auth.Credential
	if err := json.Unmarshal(bytes.TrimSpace(data), &credential); err != nil {
		return auth.Credential{}, errors.New("invalid credential record in macOS Keychain")
	}
	return credential, nil
}

func (s *Store) Clear(ctx context.Context, host string) error {
	_, err := s.runner.Run(ctx, nil, "delete-generic-password", "-s", service, "-a", host)
	if err != nil && !isMissingItem(err) {
		return errors.New("could not clear credentials from macOS Keychain")
	}

	return nil
}

func isMissingItem(err error) bool {
	var exitError *exec.ExitError
	return errors.As(err, &exitError) && exitError.ExitCode() == 44
}
