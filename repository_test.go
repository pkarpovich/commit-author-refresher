package main

import (
	"testing"
)

func TestExtractUsernameFromRepoURL(t *testing.T) {
	tests := []struct {
		repoURL  string
		expected string
	}{
		{"https://github.com/username/repo.git", "username"},
		{"https://github.com/someuser/another-repo.git", "someuser"},
	}

	for _, test := range tests {
		result, err := extractUsernameFromRepoURL(test.repoURL)
		if err != nil {
			t.Fatalf("failed to extract username from repository URL: %v", err)
		}

		if result != test.expected {
			t.Errorf("expected username '%s', got '%s'", test.expected, result)
		}
	}
}

func TestPrepareExcludedEmails(t *testing.T) {
	ctx := &RepositoryContext{
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
		t.Errorf("expected excluded emails string '%s', got '%s'", expected, result)
	}
}
