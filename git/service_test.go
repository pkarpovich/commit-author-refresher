package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewService(t *testing.T) {
	service := NewService(0)
	defaultService, ok := service.(*DefaultService)
	if !ok {
		t.Fatalf("Expected DefaultService, got %T", service)
	}
	if defaultService.Timeout != 2*time.Minute {
		t.Errorf("Expected default timeout of 2 minutes, got %v", defaultService.Timeout)
	}

	customTimeout := 5 * time.Minute
	service = NewService(customTimeout)
	defaultService, ok = service.(*DefaultService)
	if !ok {
		t.Fatalf("Expected DefaultService, got %T", service)
	}
	if defaultService.Timeout != customTimeout {
		t.Errorf("Expected custom timeout of %v, got %v", customTimeout, defaultService.Timeout)
	}
}

func TestRunCommand(t *testing.T) {
	if os.Getenv("SKIP_GIT_TESTS") != "" {
		t.Skip("Skipping git tests")
	}

	tempDir, err := os.MkdirTemp("", "git-service-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	service := NewService(0)

	err = service.RunCommand(tempDir, "init")
	if err != nil {
		t.Fatalf("RunCommand git init failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tempDir, ".git")); os.IsNotExist(err) {
		t.Errorf("Expected .git directory to exist, but it doesn't")
	}
}

func TestRunCommandWithOutput(t *testing.T) {
	if os.Getenv("SKIP_GIT_TESTS") != "" {
		t.Skip("Skipping git tests")
	}

	tempDir, err := os.MkdirTemp("", "git-service-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	service := NewService(0)
	err = service.RunCommand(tempDir, "init")
	if err != nil {
		t.Fatalf("RunCommand git init failed: %v", err)
	}

	output, err := service.RunCommandWithOutput(tempDir, "status")
	if err != nil {
		t.Fatalf("RunCommandWithOutput git status failed: %v", err)
	}

	if !strings.Contains(output, "No commits yet") && !strings.Contains(output, "On branch") {
		t.Errorf("Expected output to contain 'No commits yet' or 'On branch', got: %s", output)
	}
}
