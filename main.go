package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/pkarpovich/commit-author-refresher/repository"
)

type options struct {
	ConfigFile string `short:"f" long:"file" description:"Configuration file" default:"caf-config.json"`
	Project    string `short:"p" long:"project" description:"Run only for the specified project"`
}

func main() {
	opts := parseFlags()
	repositories, err := readConfig(opts.ConfigFile)
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	if opts.Project != "" {
		repo, err := findRepository(repositories, opts.Project)
		if err != nil {
			log.Fatalf("Project %q not found: %v", opts.Project, err)
		}
		refresher := repository.Refresher{Repo: *repo}
		if err := refresher.ProcessRepository(); err != nil {
			log.Fatalf("Failed to process repository %q: %v", opts.Project, err)
		}
		return
	}

	for _, repo := range repositories {
		refresher := repository.Refresher{Repo: repo}
		if err := refresher.ProcessRepository(); err != nil {
			log.Fatalf("Failed to process repository %q: %v", repo.Name, err)
		}
	}
}

func parseFlags() options {
	var opts options
	parser := flags.NewParser(&opts, flags.PrintErrors|flags.PassDoubleDash|flags.HelpFlag)
	if _, err := parser.Parse(); err != nil {
		var flagsErr *flags.Error
		if errors.As(err, &flagsErr) && errors.Is(flagsErr.Type, flags.ErrHelp) {
			os.Exit(2)
		}
		os.Exit(1)
	}
	return opts
}

func readConfig(configFile string) ([]repository.Repository, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file %q: %w", configFile, err)
	}

	var repositories []repository.Repository
	if err = json.Unmarshal(data, &repositories); err != nil {
		return nil, fmt.Errorf("failed to parse configuration file %q: %w", configFile, err)
	}
	return repositories, nil
}

func findRepository(repos []repository.Repository, name string) (*repository.Repository, error) {
	for i := range repos {
		if repos[i].Name == name {
			return &repos[i], nil
		}
	}
	return nil, fmt.Errorf("repository with name %q not found", name)
}
