package repository

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	remoteName      = "old-repo"
	remoteRefPrefix = "refs/remotes/" + remoteName
)

const gitFilterRepoScriptTemplate = `
if commit.committer_email not in [%s]:
    commit.committer_name = b"%s"
    commit.committer_email = b"%s"

if commit.author_email not in [%s]:
    commit.author_name = b"%s"
    commit.author_email = b"%s"

commit.message = commit.message.replace(b"%s", b"%s")
`

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

type Refresher struct {
	Repo Repository
}

func (r *Refresher) ProcessRepository() error {
	tempRepoDir, err := prepareTempFolder()
	if err != nil {
		return err
	}
	defer func() {
		removeErr := os.RemoveAll(tempRepoDir)
		if removeErr != nil {
			log.Printf("Failed to remove temporary directory: %v\n", err)
		}
	}()

	if err := r.initOriginalRepo(tempRepoDir); err != nil {
		return err
	}
	if err := r.updateCommits(tempRepoDir); err != nil {
		return err
	}
	if err := r.pushChangesToTargetRepo(tempRepoDir); err != nil {
		return err
	}

	log.Printf("Updated commits have been pushed to the target repository: %s\n", r.Repo.TargetRepo)
	return nil
}

func prepareTempFolder() (string, error) {
	tempRepoDir, err := os.MkdirTemp("", "commit-author-refresher-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %v", err)
	}
	return tempRepoDir, nil
}

func runCommand(dir string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command git failed: %v", err)
	}
	return nil
}

func runCommandWithOutput(dir, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("command %s failed: %v", name, err)
	}
	return out.String(), nil
}

func extractOwnerFromRepoURL(repoURL string) (string, error) {
	if strings.HasPrefix(repoURL, "http://") || strings.HasPrefix(repoURL, "https://") {
		u, err := url.Parse(repoURL)
		if err != nil {
			return "", fmt.Errorf("failed to parse URL: %v", err)
		}
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) < 2 {
			return "", fmt.Errorf("unexpected URL format: %s", repoURL)
		}
		return parts[0], nil
	}

	if strings.HasPrefix(repoURL, "git@") {
		parts := strings.SplitN(repoURL, ":", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("unexpected SSH URL format: %s", repoURL)
		}

		pathParts := strings.Split(strings.Trim(parts[1], "/"), "/")
		if len(pathParts) < 2 {
			return "", fmt.Errorf("unexpected SSH URL format: %s", repoURL)
		}
		return pathParts[0], nil
	}

	return "", fmt.Errorf("unsupported URL format: %s", repoURL)
}

func (r *Refresher) initBranches(dir string) error {
	branchesOut, err := runCommandWithOutput(dir, "git", "for-each-ref", "--format=%(refname:short)", remoteRefPrefix)
	if err != nil {
		return fmt.Errorf("error getting branches: %v", err)
	}

	branches := strings.Split(branchesOut, "\n")
	for _, branch := range branches {
		branch = strings.TrimSpace(branch)
		if branch == "" {
			continue
		}

		if branch == remoteName || branch == remoteName+"/HEAD" {
			continue
		}

		if !strings.Contains(branch, "/") {
			continue
		}

		branchName := strings.TrimPrefix(branch, remoteName+"/")
		if branchName != "" {
			if err := runCommand(dir, "checkout", "-b", branchName, branch); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Refresher) initOriginalRepo(dir string) error {
	if err := runCommand(dir, "init"); err != nil {
		return err
	}
	if err := runCommand(dir, "remote", "add", "old-repo", r.Repo.OriginalRepo); err != nil {
		return err
	}
	if err := runCommand(dir, "fetch", "old-repo"); err != nil {
		return err
	}
	return r.initBranches(dir)
}

func (r *Refresher) prepareExcludedEmails() string {
	excludedAuthors := append(r.Repo.ExcludedAuthors, r.Repo.Author.Email)
	var formatted []string
	for _, email := range excludedAuthors {
		formatted = append(formatted, fmt.Sprintf("b'%s'", email))
	}
	return strings.Join(formatted, ",")
}

func (r *Refresher) updateCommits(dir string) error {
	originalUser, err := extractOwnerFromRepoURL(r.Repo.OriginalRepo)
	if err != nil {
		return fmt.Errorf("error extracting username from originalRepo: %v", err)
	}

	targetUser, err := extractOwnerFromRepoURL(r.Repo.TargetRepo)
	if err != nil {
		return fmt.Errorf("error extracting username from targetRepo: %v", err)
	}

	excludedEmailsString := r.prepareExcludedEmails()

	script := fmt.Sprintf(gitFilterRepoScriptTemplate, excludedEmailsString, r.Repo.Author.Name, r.Repo.Author.Email,
		excludedEmailsString, r.Repo.Author.Name, r.Repo.Author.Email,
		originalUser, targetUser,
	)

	return runCommand(dir, "filter-repo", "--force", "--commit-callback", script)
}

func (r *Refresher) pushChangesToTargetRepo(dir string) error {
	if err := runCommand(dir, "remote", "add", "target-repo", r.Repo.TargetRepo); err != nil {
		return err
	}
	return runCommand(dir, "push", "--all", "--force", "target-repo")
}
