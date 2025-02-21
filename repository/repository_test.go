package repository

import (
	"log"
	"os"
	"strings"
	"testing"

	"github.com/pkarpovich/commit-author-refresher/git"
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
			owner, err := ExtractOwnerFromRepoURL(tc.repoURL)
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
		mockGit := &git.MockService{}

		r := &Refresher{
			Repo: Repository{
				Author: Author{
					Email: "author@example.com",
				},
				ExcludedAuthors: []string{"exclude1@example.com", "exclude2@example.com"},
			},
			GitService: mockGit,
			Logger:     log.New(os.Stderr, "[test] ", log.LstdFlags),
		}

		expected := "b'exclude1@example.com',b'exclude2@example.com',b'author@example.com'"
		result := r.prepareExcludedEmails()

		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("Empty ExcludedAuthors", func(t *testing.T) {
		mockGit := &git.MockService{}

		r := &Refresher{
			Repo: Repository{
				Author: Author{
					Email: "author@example.com",
				},
				ExcludedAuthors: []string{},
			},
			GitService: mockGit,
			Logger:     log.New(os.Stderr, "[test] ", log.LstdFlags),
		}

		expected := "b'author@example.com'"
		result := r.prepareExcludedEmails()

		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

func TestInitBranches(t *testing.T) {
	mockGit := &git.MockService{
		RunCommandWithOutputFunc: func(dir string, args ...string) (string, error) {
			if strings.Contains(strings.Join(args, " "), "for-each-ref") {
				return "old-repo/main\nold-repo/develop\nold-repo/HEAD\nold-repo\n", nil
			}
			return "", nil
		},
	}

	r := &Refresher{
		Repo: Repository{
			Name: "test-repo",
		},
		GitService: mockGit,
		Logger:     log.New(os.Stderr, "[test] ", log.LstdFlags),
	}

	err := r.initBranches("/tmp")
	if err != nil {
		t.Fatalf("initBranches failed: %v", err)
	}

	expectedCalls := []string{
		"RunCommandWithOutput:for-each-ref --format=%(refname:short) refs/remotes/old-repo",
		"RunCommand:checkout -b main old-repo/main",
		"RunCommand:checkout -b develop old-repo/develop",
	}

	for _, expected := range expectedCalls {
		found := false
		for _, call := range mockGit.CallLog {
			if call == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected call not found: %s", expected)
		}
	}
}

func TestPrepareTempFolder(t *testing.T) {
	dir, err := PrepareTempFolder()
	if err != nil {
		t.Fatalf("PrepareTempFolder failed: %v", err)
	}

	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("temp directory does not exist or is not a directory: %s", dir)
	}

	err = os.RemoveAll(dir)
	if err != nil {
		t.Fatalf("failed to remove temp dir: %v", err)
	}
}
