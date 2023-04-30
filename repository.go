package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type Author struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Repository struct {
	OriginalRepo    string   `json:"originalRepo"`
	TargetRepo      string   `json:"targetRepo"`
	Author          Author   `json:"author"`
	ExcludedAuthors []string `json:"excludedAuthors"`
}

func processRepository(repo Repository) {
	tempRepoDir, err := prepareTempFolder()
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tempRepoDir)

	initOriginalRepo(repo)
	updateCommits(repo)
	pushChangesToTargetRepo(repo)

	log.Printf("Updated commits have been pushed to the target repository: %s\n", repo.TargetRepo)
}

func prepareTempFolder() (string, error) {
	tempRepoDir, err := os.MkdirTemp("", "commit-author-refresher-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %v", err)
	}

	err = os.Chdir(tempRepoDir)
	if err != nil {
		return "", fmt.Errorf("failed to change directory to temporary directory: %v", err)
	}

	return tempRepoDir, nil
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

func runCommandWithOutput(name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return out.String(), nil
}

func initBranches() {
	branchesOut, err := runCommandWithOutput("git", "branch", "-r")
	if err != nil {
		log.Fatalf("error getting branches: %v", err)
	}

	branches := strings.Split(branchesOut, "\n")
	for _, branch := range branches {
		branch = strings.TrimSpace(strings.Replace(branch, "old-repo/", "", 1))
		if branch != "" && branch != "HEAD" {
			runCommand("git", "checkout", "-b", branch, "old-repo/"+branch)
		}
	}
}

func initOriginalRepo(repo Repository) {
	runCommand("git", "init")
	runCommand("git", "remote", "add", "old-repo", repo.OriginalRepo)
	runCommand("git", "fetch", "old-repo")

	initBranches()
}

func extractUsernameFromRepoURL(repoURL string) (string, error) {
	re := regexp.MustCompile(`https://github\.com/([^/]+)/[^/]+\.git`)
	matches := re.FindStringSubmatch(repoURL)
	if len(matches) == 0 {
		return "", fmt.Errorf("failed to extract username from repository URL: %s", repoURL)
	}
	return matches[1], nil
}

func prepareExcludedEmails(repo Repository) string {
	excludedAuthors := append(repo.ExcludedAuthors, repo.Author.Email)

	excludedAuthorsBytes := make([]string, len(excludedAuthors))
	for i, email := range excludedAuthors {
		excludedAuthorsBytes[i] = "b'" + email + "'"
	}

	return strings.Join(excludedAuthorsBytes, ",")
}

func updateCommits(repo Repository) {
	originalUser, err := extractUsernameFromRepoURL(repo.OriginalRepo)
	if err != nil {
		log.Fatalf("error extracting username from originalRepo: %v", err)
	}

	targetUser, err := extractUsernameFromRepoURL(repo.TargetRepo)
	if err != nil {
		log.Fatalf("error extracting username from targetRepo: %v", err)
	}

	excludedEmailsString := prepareExcludedEmails(repo)

	runCommand("git", "filter-repo", "--force", "--commit-callback",
		fmt.Sprintf(`
			if commit.committer_email not in [%s]:
				commit.committer_name = b"%s"
				commit.committer_email = b"%s"

			if commit.author_email not in [%s]:
				commit.author_name = b"%s"
				commit.author_email = b"%s"

			commit.message = commit.message.replace(b"%s", b"%s")
			`,
			excludedEmailsString, repo.Author.Name, repo.Author.Email,
			excludedEmailsString, repo.Author.Name, repo.Author.Email,
			originalUser, targetUser,
		),
	)
}

func pushChangesToTargetRepo(repo Repository) {
	runCommand("git", "remote", "add", "target-repo", repo.TargetRepo)
	runCommand("git", "push", "--all", "--force", "target-repo")
}
