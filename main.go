package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/pkarpovich/commit-author-refresher/config"
	"github.com/pkarpovich/commit-author-refresher/git"
	"github.com/pkarpovich/commit-author-refresher/repository"
)

type options struct {
	ConfigFile string
	Project    string
	Verbose    bool
	Timeout    time.Duration
}

func main() {
	logger := log.New(os.Stdout, "[caf] ", log.LstdFlags)

	opts := parseFlags()

	if !opts.Verbose {
		logger.SetOutput(os.Stderr)
	}

	gitService := git.NewService(opts.Timeout)
	validator := &config.DefaultValidator{}

	repositories, err := config.ReadConfig(opts.ConfigFile, validator)
	if err != nil {
		logger.Fatalf("Failed to read config: %v", err)
	}

	if opts.Project != "" {
		repo, err := config.FindRepository(repositories, opts.Project)
		if err != nil {
			logger.Fatalf("Project %q not found: %v", opts.Project, err)
		}

		refresher := repository.NewRefresher(*repo, gitService, logger)
		if err := refresher.ProcessRepository(); err != nil {
			logger.Fatalf("Failed to process repository %q: %v", opts.Project, err)
		}

		logger.Printf("Successfully processed repository: %s", opts.Project)
		return
	}

	for _, repo := range repositories {
		logger.Printf("Processing repository: %s", repo.Name)

		refresher := repository.NewRefresher(repo, gitService, logger)
		if err := refresher.ProcessRepository(); err != nil {
			logger.Fatalf("Failed to process repository %q: %v", repo.Name, err)
		}

		logger.Printf("Successfully processed repository: %s", repo.Name)
	}

	logger.Println("All repositories processed successfully")
}

func parseFlags() options {
	var opts options

	flag.StringVar(&opts.ConfigFile, "f", "caf-config.json", "Configuration file")
	flag.StringVar(&opts.ConfigFile, "file", "caf-config.json", "Configuration file")

	flag.StringVar(&opts.Project, "p", "", "Run only for the specified project")
	flag.StringVar(&opts.Project, "project", "", "Run only for the specified project")

	flag.BoolVar(&opts.Verbose, "v", false, "Enable verbose output")
	flag.BoolVar(&opts.Verbose, "verbose", false, "Enable verbose output")

	flag.DurationVar(&opts.Timeout, "t", 2*time.Minute, "Command execution timeout")
	flag.DurationVar(&opts.Timeout, "timeout", 2*time.Minute, "Command execution timeout")

	flag.Parse()

	return opts
}
