package repository

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/pkarpovich/commit-author-refresher/git"
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

type Processor interface {
	ProcessRepository() error
}

type Refresher struct {
	Repo       Repository
	GitService git.Service
	Logger     *log.Logger
}

func NewRefresher(repo Repository, gitService git.Service, logger *log.Logger) Processor {
	if logger == nil {
		logger = log.New(os.Stdout, "[repository] ", log.LstdFlags)
	}

	return &Refresher{
		Repo:       repo,
		GitService: gitService,
		Logger:     logger,
	}
}

func (r *Refresher) ProcessRepository() error {
	r.Logger.Printf("Processing repository: %s", r.Repo.Name)

	tempRepoDir, err := PrepareTempFolder()
	if err != nil {
		return err
	}
	defer func() {
		removeErr := os.RemoveAll(tempRepoDir)
		if removeErr != nil {
			r.Logger.Printf("Failed to remove temporary directory: %v", removeErr)
		}
	}()

	if err := r.initOriginalRepo(tempRepoDir); err != nil {
		return fmt.Errorf("failed to initialize original repository: %w", err)
	}

	if err := r.updateCommits(tempRepoDir); err != nil {
		return fmt.Errorf("failed to update commits: %w", err)
	}

	if err := r.pushChangesToTargetRepo(tempRepoDir); err != nil {
		return fmt.Errorf("failed to push changes to target repository: %w", err)
	}

	r.Logger.Printf("Updated commits have been pushed to the target repository: %s", r.Repo.TargetRepo)
	return nil
}

func PrepareTempFolder() (string, error) {
	tempRepoDir, err := os.MkdirTemp("", "commit-author-refresher-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}
	return tempRepoDir, nil
}

func ExtractOwnerFromRepoURL(repoURL string) (string, error) {
	if strings.HasPrefix(repoURL, "http://") || strings.HasPrefix(repoURL, "https://") {
		u, err := url.Parse(repoURL)
		if err != nil {
			return "", fmt.Errorf("failed to parse URL: %w", err)
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
			return "", fmt.Errorf("unexpected SSH URL path format: %s", repoURL)
		}
		return pathParts[0], nil
	}

	return "", fmt.Errorf("unsupported URL format: %s", repoURL)
}

func (r *Refresher) initBranches(dir string) error {
	r.Logger.Printf("Initializing branches in directory: %s", dir)

	branchesOut, err := r.GitService.RunCommandWithOutput(dir, "for-each-ref", "--format=%(refname:short)", remoteRefPrefix)
	if err != nil {
		return fmt.Errorf("error getting branches: %w", err)
	}

	branches := strings.Split(branchesOut, "\n")
	for _, branch := range branches {
		branch = strings.TrimSpace(branch)
		if branch == "" || branch == remoteName || branch == remoteName+"/HEAD" || !strings.Contains(branch, "/") {
			continue
		}

		branchName := strings.TrimPrefix(branch, remoteName+"/")
		if branchName != "" {
			r.Logger.Printf("Creating branch: %s from %s", branchName, branch)
			if err := r.GitService.RunCommand(dir, "checkout", "-b", branchName, branch); err != nil {
				return fmt.Errorf("failed to checkout branch %s: %w", branchName, err)
			}
			r.Logger.Printf("Created branch: %s", branchName)
		}
	}
	return nil
}

func (r *Refresher) initOriginalRepo(dir string) error {
	if err := r.GitService.RunCommand(dir, "init"); err != nil {
		return err
	}

	if err := r.GitService.RunCommand(dir, "remote", "add", "old-repo", r.Repo.OriginalRepo); err != nil {
		return err
	}

	if err := r.GitService.RunCommand(dir, "fetch", "old-repo"); err != nil {
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
	originalUser, err := ExtractOwnerFromRepoURL(r.Repo.OriginalRepo)
	if err != nil {
		return fmt.Errorf("error extracting username from originalRepo: %w", err)
	}

	targetUser, err := ExtractOwnerFromRepoURL(r.Repo.TargetRepo)
	if err != nil {
		return fmt.Errorf("error extracting username from targetRepo: %w", err)
	}

	excludedEmailsString := r.prepareExcludedEmails()

	script := fmt.Sprintf(gitFilterRepoScriptTemplate,
		excludedEmailsString, r.Repo.Author.Name, r.Repo.Author.Email,
		excludedEmailsString, r.Repo.Author.Name, r.Repo.Author.Email,
		originalUser, targetUser,
	)

	return r.GitService.RunCommand(dir, "filter-repo", "--force", "--commit-callback", script)
}

func (r *Refresher) pushChangesToTargetRepo(dir string) error {
	if err := r.GitService.RunCommand(dir, "remote", "add", "target-repo", r.Repo.TargetRepo); err != nil {
		return err
	}
	return r.GitService.RunCommand(dir, "push", "--all", "--force", "target-repo")
}
