package repository

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
	Name            string   `json:"name"`
	OriginalRepo    string   `json:"originalRepo"`
	TargetRepo      string   `json:"targetRepo"`
	Author          Author   `json:"author"`
	ExcludedAuthors []string `json:"excludedAuthors"`
}

type RepositoryContext struct {
	Repo Repository
}

func (ctx *RepositoryContext) ProcessRepository() {
	tempRepoDir, err := prepareTempFolder()
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tempRepoDir)

	ctx.initOriginalRepo()
	ctx.updateCommits()
	ctx.pushChangesToTargetRepo()

	log.Printf("Updated commits have been pushed to the target repository: %s\n", ctx.Repo.TargetRepo)
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

func extractUsernameFromRepoURL(repoURL string) (string, error) {
	re := regexp.MustCompile(`https://github\.com/([^/]+)/[^/]+\.git`)
	matches := re.FindStringSubmatch(repoURL)
	if len(matches) == 0 {
		return "", fmt.Errorf("failed to extract username from repository URL: %s", repoURL)
	}
	return matches[1], nil
}

func (ctx *RepositoryContext) initBranches() {
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

func (ctx *RepositoryContext) initOriginalRepo() {
	runCommand("git", "init")
	runCommand("git", "remote", "add", "old-repo", ctx.Repo.OriginalRepo)
	runCommand("git", "fetch", "old-repo")

	ctx.initBranches()
}

func (ctx *RepositoryContext) prepareExcludedEmails() string {
	excludedAuthors := append(ctx.Repo.ExcludedAuthors, ctx.Repo.Author.Email)

	excludedAuthorsBytes := make([]string, len(excludedAuthors))
	for i, email := range excludedAuthors {
		excludedAuthorsBytes[i] = "b'" + email + "'"
	}

	return strings.Join(excludedAuthorsBytes, ",")
}

func (ctx *RepositoryContext) updateCommits() {
	originalUser, err := extractUsernameFromRepoURL(ctx.Repo.OriginalRepo)
	if err != nil {
		log.Fatalf("error extracting username from originalRepo: %v", err)
	}

	targetUser, err := extractUsernameFromRepoURL(ctx.Repo.TargetRepo)
	if err != nil {
		log.Fatalf("error extracting username from targetRepo: %v", err)
	}

	excludedEmailsString := ctx.prepareExcludedEmails()

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
			excludedEmailsString, ctx.Repo.Author.Name, ctx.Repo.Author.Email,
			excludedEmailsString, ctx.Repo.Author.Name, ctx.Repo.Author.Email,
			originalUser, targetUser,
		),
	)
}

func (ctx *RepositoryContext) pushChangesToTargetRepo() {
	runCommand("git", "remote", "add", "target-repo", ctx.Repo.TargetRepo)
	runCommand("git", "push", "--all", "--force", "target-repo")
}
