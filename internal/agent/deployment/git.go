// git operations
package deployment

import (
	"fmt"

	gogit "github.com/go-git/go-git/v5"
)

func DeploymentHash(p string) (string, error) {
	repo, err := gogit.PlainOpen(p)
	if err != nil {
		return "", fmt.Errorf("Failed to open repository: %v", err)
	}
	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("Failed to get HEAD: %v", err)
	}
	return head.Hash().String(), nil
}

