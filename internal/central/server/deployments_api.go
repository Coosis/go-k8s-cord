// Central deployment file management apis poor implementation
// Look for "deployments_page.go" for the htmx integration and html templates.
package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"

	. "github.com/Coosis/go-k8s-cord/internal/central/deployment"
	. "github.com/Coosis/go-k8s-cord/internal/central/model"

	log "github.com/sirupsen/logrus"
)

const (
	DEPLOYMENTS_PATH = "/api/v1/deployments"
	DEPLOYMENTS_UPLOAD_PATH = "/api/v1/deployments/upload"
	DEPLOYMENTS_DELETE_PATH = "/api/v1/deployments/delete"
)

func(s *CentralServer) setupDeploymentRoutes() {
	// List deployments
	s.HandleFunc(DEPLOYMENTS_PATH, s.ListDeployments)

	// Upload deployment file
	s.HandleFunc(DEPLOYMENTS_UPLOAD_PATH, s.UploadDeploymentFile)

	// Delete deployment file
	s.HandleFunc(DEPLOYMENTS_DELETE_PATH, s.DeleteDeploymentFile)
}

func(s *CentralServer) ListDeployments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Warn("ListDeployments called with method: ", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	deployments, err := ListDeploymentFiles(s.repo)
	if err != nil {
		log.Error("Failed to list deployments: ", err)
		http.Error(w, "Failed to list deployments", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(deployments)
	if err != nil {
		log.Error("Failed to encode deployments: ", err)
		http.Error(w, "Failed to encode deployments", http.StatusInternalServerError)
		return
	}
}

func(s *CentralServer) UploadDeploymentFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Warn("UploadDeploymentFile called with method: ", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg := GetCentralConfig()

	const maxMemory = 32 << 20 // 32MB
    if err := r.ParseMultipartForm(maxMemory); err != nil {
		log.Error("Failed to parse multipart form: ", err)
        http.Error(w, "Failed to parse multipart form: "+err.Error(), http.StatusBadRequest)
        return
    }

	uploadedFiles := r.MultipartForm.File["files"]
    if len(uploadedFiles) == 0 {
		log.Warn("No files provided in the upload request")
        http.Error(w, "No files provided", http.StatusBadRequest)
        return
    }

    for _, header := range uploadedFiles {
        srcFile, err := header.Open()
        if err != nil {
			log.Error("Failed to open uploaded file: ", err)
            http.Error(w, "Failed to open uploaded file: "+err.Error(), http.StatusInternalServerError)
            return
        }
        defer srcFile.Close()

        dstPath := filepath.Join(cfg.DeploymentsDir, header.Filename)
		log.Infof("Saving file to: {%s}", dstPath)
        dstFile, err := os.Create(dstPath)
        if err != nil {
			log.Error("Failed to create file on server: ", err)
            http.Error(w, "Failed to create file on server: "+err.Error(), http.StatusInternalServerError)
            return
        }
        defer dstFile.Close()

        if _, err := io.Copy(dstFile, srcFile); err != nil {
			log.Error("Failed to save file: ", err)
            http.Error(w, "Failed to save file: "+err.Error(), http.StatusInternalServerError)
            return
		}

		err = AddFile(s.repo, header.Filename)
		if err != nil {
			log.Error("Failed to add file to git repository: ", err)
			http.Error(w, "Failed to add file to git repository: "+err.Error(), http.StatusInternalServerError)
			return
		}
    }

	// git operations
	err := CommitChanges(s.repo, "Uploaded deployment files")
	if err != nil {
		log.Error("Failed to commit changes to git repository: ", err)
		http.Error(w, "Failed to commit changes to git repository: " + err.Error(), http.StatusInternalServerError)
		return
	}
	err = PushChanges(s.repo, cfg.GitRemoteName)
	if err != nil {
		log.Error("Failed to push changes to git repository: ", err)
		http.Error(w, "Failed to push changes to git repository: " + err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("File uploaded successfully"))
}

func(s *CentralServer) DeleteDeploymentFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Error("Failed to parse form: ", err)
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}
	cfg := GetCentralConfig()

	q := r.URL.Query()
	filenames := q["filename"]
	if len(filenames) == 0 {
		log.Warn("Filename is required for deletion")
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	for _, filename := range filenames {
		if filename == "" {
			continue // Skip empty filenames
		}

		log.Infof("Deleting file: %s", filename)

		fp := filepath.Join(cfg.DeploymentsDir, filename)
		if _, err := os.Stat(fp); os.IsNotExist(err) {
			log.Warn("File does not exist: ", fp)
			http.Error(w, "File does not exist: "+filename, http.StatusNotFound)
			return
		}

		if err := os.Remove(fp); err != nil {
			log.Error("Failed to delete file: ", err)
			http.Error(w, "Failed to delete file: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Infof("Deleted file: %s", filename)

		err := AddFile(s.repo, filename)
		if err != nil {
			log.Error("Failed to add file to git repository: ", err)
			http.Error(w, "Failed to add file to git repository: "+err.Error(), http.StatusInternalServerError)
			return
		}
		err = CommitChanges(s.repo, "Deleted deployment file: " + filename)
		if err != nil {
			log.Error("Failed to commit changes to git repository: ", err)
			http.Error(w, "Failed to commit changes to git repository: " + err.Error(), http.StatusInternalServerError)
			return
		}
		err = PushChanges(s.repo, cfg.GitRemoteName)
		if err != nil {
			log.Error("Failed to push changes to git repository: ", err)
			http.Error(w, "Failed to push changes to git repository: " + err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("File deleted successfully"))
}
