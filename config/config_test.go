package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkarpovich/commit-author-refresher/repository"
)

type MockValidator struct {
	ValidateFunc func(repo repository.Repository) error
	CallCount    int
}

func (m *MockValidator) Validate(repo repository.Repository) error {
	m.CallCount++
	if m.ValidateFunc != nil {
		return m.ValidateFunc(repo)
	}
	return nil
}

func TestDefaultValidator_Validate(t *testing.T) {
	tests := []struct {
		name      string
		repo      repository.Repository
		expectErr bool
	}{
		{
			name: "Valid Repository",
			repo: repository.Repository{
				Name:         "valid-repo",
				OriginalRepo: "https://github.com/user/repo.git",
				TargetRepo:   "https://github.com/user/target-repo.git",
				Author: repository.Author{
					Name:  "Test User",
					Email: "test@example.com",
				},
			},
			expectErr: false,
		},
		{
			name: "Missing Name",
			repo: repository.Repository{
				OriginalRepo: "https://github.com/user/repo.git",
				TargetRepo:   "https://github.com/user/target-repo.git",
				Author: repository.Author{
					Name:  "Test User",
					Email: "test@example.com",
				},
			},
			expectErr: true,
		},
		{
			name: "Missing Original Repo",
			repo: repository.Repository{
				Name:       "invalid-repo",
				TargetRepo: "https://github.com/user/target-repo.git",
				Author: repository.Author{
					Name:  "Test User",
					Email: "test@example.com",
				},
			},
			expectErr: true,
		},
		{
			name: "Missing Target Repo",
			repo: repository.Repository{
				Name:         "invalid-repo",
				OriginalRepo: "https://github.com/user/repo.git",
				Author: repository.Author{
					Name:  "Test User",
					Email: "test@example.com",
				},
			},
			expectErr: true,
		},
		{
			name: "Missing Author Name",
			repo: repository.Repository{
				Name:         "invalid-repo",
				OriginalRepo: "https://github.com/user/repo.git",
				TargetRepo:   "https://github.com/user/target-repo.git",
				Author: repository.Author{
					Email: "test@example.com",
				},
			},
			expectErr: true,
		},
		{
			name: "Missing Author Email",
			repo: repository.Repository{
				Name:         "invalid-repo",
				OriginalRepo: "https://github.com/user/repo.git",
				TargetRepo:   "https://github.com/user/target-repo.git",
				Author: repository.Author{
					Name: "Test User",
				},
			},
			expectErr: true,
		},
	}

	validator := &DefaultValidator{}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.Validate(tc.repo)
			if tc.expectErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestReadConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(originalDir)

	testRepo := []repository.Repository{
		{
			Name:         "test-repo",
			OriginalRepo: "https://github.com/user/repo.git",
			TargetRepo:   "https://github.com/user/target-repo.git",
			Author: repository.Author{
				Name:  "Test User",
				Email: "test@example.com",
			},
			ExcludedAuthors: []string{"exclude@example.com"},
		},
	}

	t.Run("JSON Config", func(t *testing.T) {
		jsonFile := filepath.Join(tempDir, "config.json")
		jsonData, err := json.Marshal(testRepo)
		if err != nil {
			t.Fatalf("Failed to marshal JSON: %v", err)
		}

		if err := os.WriteFile(jsonFile, jsonData, 0644); err != nil {
			t.Fatalf("Failed to write JSON file: %v", err)
		}

		mockValidator := &MockValidator{}
		repos, err := ReadConfig(jsonFile, mockValidator)
		if err != nil {
			t.Fatalf("ReadConfig failed: %v", err)
		}

		if len(repos) != 1 {
			t.Fatalf("Expected 1 repository, got %d", len(repos))
		}

		if mockValidator.CallCount != 1 {
			t.Errorf("Expected validator to be called once, got %d", mockValidator.CallCount)
		}

		if repos[0].Name != testRepo[0].Name {
			t.Errorf("Expected repo name %q, got %q", testRepo[0].Name, repos[0].Name)
		}
	})
}

func TestFindRepository(t *testing.T) {
	repos := []repository.Repository{
		{Name: "repo1"},
		{Name: "repo2"},
		{Name: "repo3"},
	}

	t.Run("Existing Repository", func(t *testing.T) {
		repo, err := FindRepository(repos, "repo2")
		if err != nil {
			t.Fatalf("FindRepository failed: %v", err)
		}
		if repo.Name != "repo2" {
			t.Errorf("Expected repo name %q, got %q", "repo2", repo.Name)
		}
	})

	t.Run("Non-existing Repository", func(t *testing.T) {
		_, err := FindRepository(repos, "repo4")
		if err == nil {
			t.Fatal("Expected error but got nil")
		}
	})
}
