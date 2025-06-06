// helper functions for git operations
package deployment

import (
	"fmt"
	"net"
	"os"
	"slices"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func sshAuth() (*ssh.PublicKeysCallback, error) {
	conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH agent: %v", err)
	}
	agentClient := agent.NewClient(conn)

	return &ssh.PublicKeysCallback{
		User: "git",
		Callback: agentClient.Signers,
	}, nil
}

func DeploymentsHash(repo *gogit.Repository) (string, error) {
	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("Failed to get HEAD: %v", err)
	}
	return head.Hash().String(), nil
}

func AddFile(repo *gogit.Repository, filePath string) error {
	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("Failed to get worktree: %v", err)
	}

	_, err = w.Add(filePath)
	if err != nil {
		return fmt.Errorf("Failed to add file: %v", err)
	}

	return nil
}

func CommitChanges(repo *gogit.Repository, message string) error {
	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("Failed to get worktree: %v", err)
	}

	_, err = w.Commit(message, &gogit.CommitOptions{
		All: true,
	})
	if err != nil {
		return fmt.Errorf("Failed to commit changes: %v", err)
	}

	return nil
}

func PushChanges(repo *gogit.Repository, remoteName string) error {
	remote, err := repo.Remote(remoteName)
	if err != nil {
		return fmt.Errorf("Failed to get remote: %v", err)
	}

	auth, err := sshAuth()
	if err != nil {
		return fmt.Errorf("Failed to get SSH auth: %v", err)
	}

	err = remote.Push(&gogit.PushOptions{
		RemoteName: remoteName,
		Auth:        auth,
	})
	if err != nil {
		return fmt.Errorf("Failed to push changes: %v", err)
	}

	return nil
}

func ShouldIgnoreFile(filename string) bool {
	excluded := []string{
		".gitignore",
		"LICENSE",
		"README.md",
	}

	return slices.Contains(excluded, filename)
}

func ListDeploymentFiles(repo *gogit.Repository) ([]string, error) {
	files := []string{}
	head, err := repo.Head()
	if err != nil {
		return nil, err
	}
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, err
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	tree.Files().ForEach(func(f *object.File) error {
		if f.Mode.IsFile() && !ShouldIgnoreFile(f.Name) {
			files = append(files, f.Name)
		}
		return nil
	})

	return files, nil
}
