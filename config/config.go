package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkarpovich/commit-author-refresher/repository"
)

type Validator interface {
	Validate(repo repository.Repository) error
}

type DefaultValidator struct{}

func (v *DefaultValidator) Validate(repo repository.Repository) error {
	if repo.Name == "" {
		return errors.New("repository name cannot be empty")
	}
	if repo.OriginalRepo == "" {
		return errors.New("original repository URL cannot be empty")
	}
	if repo.TargetRepo == "" {
		return errors.New("target repository URL cannot be empty")
	}
	if repo.Author.Name == "" {
		return errors.New("author name cannot be empty")
	}
	if repo.Author.Email == "" {
		return errors.New("author email cannot be empty")
	}
	return nil
}

func ReadConfig(configFile string, validator Validator) ([]repository.Repository, error) {
	absPath, err := filepath.Abs(configFile)
	if err != nil {
		return nil, fmt.Errorf("invalid configuration file path %q: %w", configFile, err)
	}

	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access configuration file %q: %w", configFile, err)
	}

	if fileInfo.IsDir() {
		return nil, fmt.Errorf("configuration file %q is a directory", configFile)
	}

	ext := strings.ToLower(filepath.Ext(absPath))
	if ext != ".json" && ext != ".yaml" && ext != ".yml" {
		return nil, fmt.Errorf("configuration file must have .json, .yaml, or .yml extension")
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file %q: %w", configFile, err)
	}

	var repositories []repository.Repository

	switch ext {
	case ".json":
		if err = json.Unmarshal(data, &repositories); err != nil {
			return nil, fmt.Errorf("failed to parse JSON configuration file %q: %w", configFile, err)
		}
	}

	for i, repo := range repositories {
		if err := validator.Validate(repo); err != nil {
			return nil, fmt.Errorf("invalid configuration for repository %q: %w", repo.Name, err)
		}

		repositories[i].Author.Email = os.ExpandEnv(repo.Author.Email)
		repositories[i].Author.Name = os.ExpandEnv(repo.Author.Name)
		repositories[i].OriginalRepo = os.ExpandEnv(repo.OriginalRepo)
		repositories[i].TargetRepo = os.ExpandEnv(repo.TargetRepo)
	}

	return repositories, nil
}

func FindRepository(repos []repository.Repository, name string) (*repository.Repository, error) {
	for i := range repos {
		if repos[i].Name == name {
			return &repos[i], nil
		}
	}
	return nil, fmt.Errorf("repository with name %q not found", name)
}
