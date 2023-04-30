package main

import (
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

	//replaceOriginalUser, err := getReplaceOriginalUsernameString(repo)
	//if err != nil {
	//	log.Fatal(err)
	//}

	excludedEmailsString := prepareExcludedEmails(repo)

	runCommand("git", "filter-repo", "--force", "--commit-callback",
		fmt.Sprintf(`
			if commit.committer_email not in [%s]:
				commit.committer_name = b"%s"
				commit.committer_email = b"%s"
			if commit.author_email not in [%s]:
				commit.author_name = b"%s"
				commit.author_email = b"%s"
			`,
			excludedEmailsString, newAuthorName, newAuthorEmail, excludedEmailsString, newAuthorName, newAuthorEmail,
		),
	)

	runCommand("git", "remote", "add", "target-repo", targetRepo)

	runCommand("git", "push", "--all", "--force", "target-repo")

	fmt.Printf("Updated commits have been pushed to the target repository: %s\n", targetRepo)
}

func getReplaceOriginalUsernameString(repo Repository) (string, error) {
	originalUser, err := extractUsernameFromRepoURL(repo.OriginalRepo)
	if err != nil {
		return "", fmt.Errorf("error extracting username from originalRepo: %v", err)
	}

	targetUser, err := extractUsernameFromRepoURL(repo.TargetRepo)
	if err != nil {
		return "", fmt.Errorf("error extracting username from targetRepo: %v", err)
	}

	// Add the following line before the git filter-branch command
	replaceOriginalUser := fmt.Sprintf("export GIT_MSG=$(echo \"$GIT_MSG\" | sed 's/%s/%s/g');", originalUser, targetUser)

	return replaceOriginalUser, nil
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

func extractUsernameFromRepoURL(repoURL string) (string, error) {
	re := regexp.MustCompile(`https:\/\/github\.com\/([^/]+)\/[^/]+\.git`)
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
