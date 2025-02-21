package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Service interface {
	RunCommand(dir string, args ...string) error
	RunCommandWithOutput(dir string, args ...string) (string, error)
}

type MockService struct {
	RunCommandFunc           func(dir string, args ...string) error
	RunCommandWithOutputFunc func(dir string, args ...string) (string, error)
	CallLog                  []string
}

func (m *MockService) EnsureCallLog() {
	if m.CallLog == nil {
		m.CallLog = []string{}
	}
}

func (m *MockService) RunCommand(dir string, args ...string) error {
	m.EnsureCallLog()
	m.CallLog = append(m.CallLog, "RunCommand:"+strings.Join(args, " "))

	if m.RunCommandFunc != nil {
		return m.RunCommandFunc(dir, args...)
	}
	return nil
}

func (m *MockService) RunCommandWithOutput(dir string, args ...string) (string, error) {
	m.EnsureCallLog()
	m.CallLog = append(m.CallLog, "RunCommandWithOutput:"+strings.Join(args, " "))

	if m.RunCommandWithOutputFunc != nil {
		return m.RunCommandWithOutputFunc(dir, args...)
	}
	return "", nil
}

type DefaultService struct {
	Timeout time.Duration
}

func NewService(timeout time.Duration) Service {
	if timeout == 0 {
		timeout = 2 * time.Minute
	}
	return &DefaultService{
		Timeout: timeout,
	}
}

func (s *DefaultService) RunCommand(dir string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), s.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git command failed: %v (args: %v)", err, args)
	}
	return nil
}

func (s *DefaultService) RunCommandWithOutput(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), s.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git command failed: %v (args: %v)", err, args)
	}
	return out.String(), nil
}
