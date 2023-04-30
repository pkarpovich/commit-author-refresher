package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/pkarpovich/commit-author-refresher/repository"
)

type options struct {
	ConfigFile string `short:"f" long:"file" description:"config file" default:"caf-config.json"`
	Project    string `short:"p" long:"project" description:"run only for the specified project"`
}

func main() {
	var opts options
	p := flags.NewParser(&opts, flags.PrintErrors|flags.PassDoubleDash|flags.HelpFlag)
	if _, err := p.Parse(); err != nil {
		if err.(*flags.Error).Type != flags.ErrHelp {
			os.Exit(1)
		}
		os.Exit(2)
	}

	data, err := os.ReadFile(opts.ConfigFile)
	if err != nil {
		log.Fatalf("Failed to read configuration file: %s", opts.ConfigFile)
	}

	var repositories []repository.Repository
	err = json.Unmarshal(data, &repositories)
	if err != nil {
		log.Fatalf("Failed to parse configuration file: %s. Error: %v", opts.ConfigFile, err)
	}

	if opts.Project != "" {
		repo := find(repositories, opts.Project)
		ctx := repository.RepositoryContext{Repo: *repo}
		ctx.ProcessRepository()
		return
	}

	for _, repo := range repositories {
		ctx := repository.RepositoryContext{Repo: repo}
		ctx.ProcessRepository()
	}
}

func find(repositories []repository.Repository, name string) *repository.Repository {
	for _, repo := range repositories {
		if repo.Name == name {
			return &repo
		}
	}

	return nil
}
