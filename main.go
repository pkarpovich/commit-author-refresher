package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

type Author struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Repository struct {
	OriginalRepo string `json:"originalRepo"`
	TargetRepo   string `json:"targetRepo"`
	Author       Author `json:"author"`
}

func main() {
	configFile := "config.json"

	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Failed to read configuration file: %s", configFile)
	}

	var repositories []Repository
	err = json.Unmarshal(data, &repositories)
	if err != nil {
		log.Fatalf("Failed to parse configuration file: %s. Error: %v", configFile, err)
	}

	for _, repo := range repositories {
		processRepository(repo)
	}
}

func processRepository(repo Repository) {
	originalRepo := repo.OriginalRepo
	targetRepo := repo.TargetRepo
	newAuthorEmail := repo.Author.Email
	newAuthorName := repo.Author.Name

	tempRepoDir, err := os.MkdirTemp("", "commit-author-refresher-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tempRepoDir)

	err = os.Chdir(tempRepoDir)
	if err != nil {
		log.Fatal(err)
	}

	runCommand("git", "init")

	runCommand("git", "remote", "add", "old-repo", originalRepo)

	runCommand("git", "fetch", "old-repo")

	branchesOut, err := exec.Command("git", "branch", "-r").Output()
	if err != nil {
		log.Fatal(err)
	}

	branches := strings.Split(string(branchesOut), "\n")
	for _, branch := range branches {
		branch = strings.TrimSpace(strings.Replace(branch, "old-repo/", "", 1))
		if branch != "" && branch != "HEAD" {
			runCommand("git", "checkout", "-b", branch, "old-repo/"+branch)
		}
	}

	runCommand("git", "filter-branch", "-f", "--env-filter",
		`export GIT_COMMITTER_NAME="`+newAuthorName+`"
         export GIT_COMMITTER_EMAIL="`+newAuthorEmail+`"
         export GIT_AUTHOR_NAME="`+newAuthorName+`"
         export GIT_AUTHOR_EMAIL="`+newAuthorEmail+`"`,
		"--tag-name-filter", "cat", "--", "--all")

	runCommand("git", "remote", "add", "target-repo", targetRepo)

	runCommand("git", "push", "--all", "--force", "target-repo")

	fmt.Printf("Updated commits have been pushed to the target repository: %s\n", targetRepo)
}

func runCommand(name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
