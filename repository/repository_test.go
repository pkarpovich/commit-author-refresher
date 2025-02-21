package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractOwnerFromRepoURL(t *testing.T) {
	tests := []struct {
		name      string
		repoURL   string
		expected  string
		expectErr bool
	}{
		{
			name:     "HTTPS GitHub",
			repoURL:  "https://github.com/username/repo.git",
			expected: "username",
		},
		{
			name:     "HTTPS Custom Host",
			repoURL:  "https://custom.gitlab.com/owner/repo.git",
			expected: "owner",
		},
		{
			name:     "HTTPS Custom Host",
			repoURL:  "https://git.example.com/owner/repo.git",
			expected: "owner",
		},
		{
			name:     "SSH GitHub",
			repoURL:  "git@github.com:username/repo.git",
			expected: "username",
		},
		{
			name:     "SSH Custom Host",
			repoURL:  "git@custom.org:owner/repo.git",
			expected: "owner",
		},
		{
			name:      "Invalid URL Format",
			repoURL:   "not-a-valid-url",
			expectErr: true,
		},
		{
			name:      "HTTPS URL with insufficient segments",
			repoURL:   "https://github.com/",
			expectErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			owner, err := extractOwnerFromRepoURL(tc.repoURL)
			if tc.expectErr {
				if err == nil {
					t.Errorf("expected error for URL %q but got none (result: %q)", tc.repoURL, owner)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for URL %q: %v", tc.repoURL, err)
				}
				if owner != tc.expected {
					t.Errorf("expected %q, got %q", tc.expected, owner)
				}
			}
		})
	}
}

func TestPrepareExcludedEmails(t *testing.T) {
	t.Run("Non-empty ExcludedAuthors", func(t *testing.T) {
		ctx := &Refresher{
			Repo: Repository{
				Author: Author{
					Email: "author@example.com",
				},
				ExcludedAuthors: []string{"exclude1@example.com", "exclude2@example.com"},
			},
		}
		expected := "b'exclude1@example.com',b'exclude2@example.com',b'author@example.com'"
		result := ctx.prepareExcludedEmails()

		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("Empty ExcludedAuthors", func(t *testing.T) {
		ctx := &Refresher{
			Repo: Repository{
				Author: Author{
					Email: "author@example.com",
				},
				ExcludedAuthors: []string{},
			},
		}
		expected := "b'author@example.com'"
		result := ctx.prepareExcludedEmails()

		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("Order of emails", func(t *testing.T) {
		ctx := &Refresher{
			Repo: Repository{
				Author: Author{
					Email: "author@example.com",
				},
				ExcludedAuthors: []string{"exclude@example.com"},
			},
		}
		expected := "b'exclude@example.com',b'author@example.com'"
		result := ctx.prepareExcludedEmails()

		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

func TestPrepareTempFolder(t *testing.T) {
	dir, err := prepareTempFolder()
	if err != nil {
		t.Fatalf("prepareTempFolder failed: %v", err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("temp directory does not exist or is not a directory: %s", dir)
	}

	err = os.RemoveAll(dir)
	if err != nil {
		t.Fatalf("failed to remove temp dir: %v", err)
	}
}

func TestRunCommandWithOutput(t *testing.T) {
	out, err := runCommandWithOutput("", "echo", "hello")
	if err != nil {
		t.Fatalf("runCommandWithOutput failed: %v", err)
	}
	if got := strings.TrimSpace(out); got != "hello" {
		t.Errorf("expected 'hello', got %q", got)
	}
}

func TestRunCommand(t *testing.T) {
	if err := runCommand("", "echo", "test"); err != nil {
		t.Fatalf("runCommand failed: %v", err)
	}
}

func TestInitOriginalRepo(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-init-original-repo-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() {
		err := os.RemoveAll(tempDir)
		if err != nil {
			t.Fatalf("failed to remove temp dir: %v", err)
		}
	}()

	bareRepoDir := filepath.Join(tempDir, "bare-repo.git")
	if err := runCommand(tempDir, "git", "init", "--bare", bareRepoDir); err != nil {
		t.Fatalf("failed to create bare repository: %v", err)
	}

	repo := Repository{
		Name:         "test-repo",
		OriginalRepo: "file://" + bareRepoDir,
		TargetRepo:   "",
		Author: Author{
			Name:  "Test Author",
			Email: "author@test.com",
		},
		ExcludedAuthors: []string{},
	}
	ctx := &Refresher{Repo: repo}

	if err := ctx.initOriginalRepo(tempDir); err != nil {
		t.Fatalf("initOriginalRepo failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tempDir, ".git")); err != nil {
		t.Errorf("expected .git directory after initialization, but got error: %v", err)
	}
}
