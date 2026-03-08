package services

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
    "github.com/go-git/go-git/v5/config"
    "github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
    memorystorage "github.com/go-git/go-git/v5/storage/memory"
)

type Cloner interface {
	CloneRepository(repoURL, outputDir string) error
	CloneRepositoryBranch(repoURL, branch, outputDir string) error
	ListRemoteBranches(repoURL string) ([]string, error)
	SetSSHKeyPath(path string)
}

type GitCloner struct {
	FileSystem FileSystem
	SSHKeyPath string // optional override; auto-detected if empty
}

func (gc *GitCloner) CloneRepository(repoURL, outputDir string) error {
	cloneOptions := &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
	}

	if gc.isSSH(repoURL) {
		sshKeyPath, err := gc.getSSHKeyPath()
		if err != nil {
			return fmt.Errorf("failed to get SSH key path: %w", err)
		}
		auth, err := ssh.NewPublicKeysFromFile("git", sshKeyPath, "")
		if err != nil {
			return fmt.Errorf("failed to create SSH auth method: %v", err)
		}
		cloneOptions.Auth = auth
	}

	_, err := git.PlainClone(outputDir, false, cloneOptions)
	if err != nil {
		return fmt.Errorf("failed to clone the repository: %v", err)
	}
	return nil
}

func (gc *GitCloner) CloneRepositoryBranch(repoURL, branch, outputDir string) error {
    cloneOptions := &git.CloneOptions{
        URL:      repoURL,
        Progress: os.Stdout,
    }

    if branch != "" {
        cloneOptions.ReferenceName = plumbing.NewBranchReferenceName(branch)
        cloneOptions.SingleBranch = true
        cloneOptions.Depth = 1
    }

    if gc.isSSH(repoURL) {
        sshKeyPath, err := gc.getSSHKeyPath()
        if err != nil {
            return fmt.Errorf("failed to get SSH key path: %w", err)
        }
        auth, err := ssh.NewPublicKeysFromFile("git", sshKeyPath, "")
        if err != nil {
            return fmt.Errorf("failed to create SSH auth method: %v", err)
        }
        cloneOptions.Auth = auth
    }

    _, err := git.PlainClone(outputDir, false, cloneOptions)
    if err != nil {
        return fmt.Errorf("failed to clone the repository: %v", err)
    }
    return nil
}

// ListRemoteBranches lists remote branch names without cloning locally.
func (gc *GitCloner) ListRemoteBranches(repoURL string) ([]string, error) {
    remote := git.NewRemote(memorystorage.NewStorage(), &config.RemoteConfig{URLs: []string{repoURL}})
    listOpts := &git.ListOptions{}

    if gc.isSSH(repoURL) {
        sshKeyPath, err := gc.getSSHKeyPath()
        if err != nil {
            return nil, fmt.Errorf("failed to get SSH key path: %w", err)
        }
        auth, err := ssh.NewPublicKeysFromFile("git", sshKeyPath, "")
        if err != nil {
            return nil, fmt.Errorf("failed to create SSH auth method: %v", err)
        }
        listOpts.Auth = auth
    }

    refs, err := remote.List(listOpts)
    if err != nil {
        return nil, err
    }
    var branches []string
    seen := map[string]struct{}{}
    for _, r := range refs {
        if r.Name().IsBranch() {
            name := r.Name().Short()
            if _, ok := seen[name]; !ok {
                seen[name] = struct{}{}
                branches = append(branches, name)
            }
        }
    }
    return branches, nil
}

// HeadSHA returns the HEAD commit SHA from a local git repository
func HeadSHA(repoPath string) (string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "", err
	}
	ref, err := repo.Head()
	if err != nil {
		return "", err
	}
	return ref.Hash().String(), nil
}

func (gc *GitCloner) SetSSHKeyPath(path string) {
	gc.SSHKeyPath = path
}

func (gc *GitCloner) isSSH(repoURL string) bool {
	return strings.HasPrefix(repoURL, "git@") || strings.HasPrefix(repoURL, "ssh://")
}

func (gc *GitCloner) getSSHKeyPath() (string, error) {
	if gc.SSHKeyPath != "" {
		if _, err := os.Stat(gc.SSHKeyPath); err != nil {
			return "", fmt.Errorf("specified SSH key not found: %s", gc.SSHKeyPath)
		}
		return gc.SSHKeyPath, nil
	}

	u, err := user.Current()
	if err != nil {
		return "", err
	}
	sshDir := filepath.Join(u.HomeDir, ".ssh")

	// Try common key types in preference order
	candidates := []string{"id_ed25519", "id_ecdsa", "id_rsa"}
	for _, name := range candidates {
		p := filepath.Join(sshDir, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("no SSH key found in %s (tried: %s)", sshDir, strings.Join(candidates, ", "))
}
