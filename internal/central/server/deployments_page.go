// Deployment file management page with htmx, look for "deployments_api.go" for the API implementation.
// For html look for deployments.html and file_entry.html in templates directory.
package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"strings"

	. "github.com/Coosis/go-k8s-cord/internal/central/model"

	log "github.com/sirupsen/logrus"
)

const (
	DEPLOYMENTS_LIST_URL = "http://localhost%s" + DEPLOYMENTS_PATH
	DEPLOYMENTS_UPLOAD_URL = "http://localhost%s" + DEPLOYMENTS_UPLOAD_PATH
	DEPLOYMENTS_DELETE_URL = "http://localhost%s" + DEPLOYMENTS_DELETE_PATH + "?%s"

	PAGE_DEPLOYMENTS_PATH = "/deployments"
	PAGE_DEPLOYMENTS_LIST_PATH = "/deployments/list"
	PAGE_DEPLOYMENTS_UPLOAD_PATH = "/deployments/upload"
	PAGE_DEPLOYMENTS_DELETE_PATH = "/deployments/delete"
)

func(s *CentralServer) setupDeploymentsHTML() {
	cfg := GetCentralConfig()
	uploadTemplate := template.Must(template.ParseFiles(
		"templates/base.html",
		"templates/deployments.html",
	))
	s.HandleFunc(PAGE_DEPLOYMENTS_PATH, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		err := uploadTemplate.Execute(w, nil)
		if err != nil {
			log.Error("Failed to execute upload template: ", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})

	noitems := template.Must(template.New("noitems").Parse(`<p>No files found.</p>`))
	listitems := template.Must(template.ParseFiles("templates/file_entry.html"))
	s.HandleFunc(PAGE_DEPLOYMENTS_LIST_PATH, func(w http.ResponseWriter, r *http.Request) {
		endpoint := fmt.Sprintf(DEPLOYMENTS_LIST_URL, cfg.HTTPSPort)
		deployments, err := http.Get(endpoint)
		if err != nil {
			log.Error("Failed to list deployments: ", err)
			http.Error(w, "Failed to list deployments: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// decode from json
		defer deployments.Body.Close()
		if deployments.StatusCode != http.StatusOK {
			log.Error("Failed to list deployments, status code: ", deployments.StatusCode)
			http.Error(w, "Failed to list deployments, status code: "+deployments.Status, deployments.StatusCode)
			return
		}
		var items []string
		decoder := json.NewDecoder(deployments.Body)
		err = decoder.Decode(&items)
		if err != nil {
			log.Error("Failed to decode deployments response: ", err)
			http.Error(w, "Failed to decode deployments response: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Debugf("Deployments list: %v", len(items))

		htmlVars := make(map[string]any)
		htmlVars["Deployments"] = items

		w.Header().Set("Content-Type", "text/html")
		if len(items) == 0 {
			err = noitems.Execute(w, nil)
		} else {
			err = listitems.Execute(w, htmlVars)
		}
		if err != nil {
			log.Error("Failed to execute template: ", err)
			http.Error(w, "Failed to execute template: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})

	s.HandleFunc(PAGE_DEPLOYMENTS_UPLOAD_PATH, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			log.Warn("Upload deployment file called with non-POST method")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		contentType := r.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "multipart/form-data") {
			log.Error("Unexpected Content-Type, expected multipart/form-data; got: ", contentType)
			http.Error(w, "Expected multipart/form-data", http.StatusBadRequest)
			return
		}

		endpoint := fmt.Sprintf(DEPLOYMENTS_UPLOAD_URL, cfg.HTTPSPort)
		proxyReq, err := http.NewRequest(http.MethodPost, endpoint, r.Body)
		if err != nil {
			log.Error("Failed to create proxy request: ", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		proxyReq.Header.Set("Content-Type", contentType)

		client := &http.Client{}
		resp, err := client.Do(proxyReq)
		if err != nil {
			log.Error("Failed to send proxy upload request: ", err)
			http.Error(w, "Failed to upload deployment file: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			log.Error("Downstream upload returned non-OK: ", resp.Status, string(bodyBytes))
			http.Error(
				w,
				fmt.Sprintf("Failed to upload deployment file: %s", string(bodyBytes)),
				resp.StatusCode,
			)
			return
		}
		http.Redirect(w, r, PAGE_DEPLOYMENTS_LIST_PATH, http.StatusSeeOther)	
	})

	s.HandleFunc(PAGE_DEPLOYMENTS_DELETE_PATH, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			log.Warn("Delete deployment file called with non-POST method")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			log.Error("Failed to parse form: ", err)
			http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
			return
		}

		filenames := r.Form["filename"]
		if len(filenames) == 0 {
			log.Warn("No filename provided for deletion")
			http.Error(w, "No filename provided", http.StatusBadRequest)
			return
		}

		// Build a URL‐encoded query string with all “filename” parameters
		q := url.Values{}
		for _, f := range filenames {
			if f == "" {
				continue
			}
			log.Debugf("Deleting deployment file: %s", f)
			q.Add("filename", f)
		}

		endpoint := fmt.Sprintf(DEPLOYMENTS_DELETE_URL, cfg.HTTPSPort, q.Encode())

		req, err := http.NewRequest(http.MethodDelete, endpoint, nil)
		if err != nil {
			log.Error("Failed to create DELETE request: ", err)
			http.Error(w, "Failed to create delete request: "+err.Error(), http.StatusInternalServerError)
			return
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Error("Failed to send DELETE request: ", err)
			http.Error(w, "Failed to send delete request: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			log.Error("Delete handler returned non-OK status: ", resp.Status, string(bodyBytes))
			http.Error(
				w,
				fmt.Sprintf("Failed to delete deployment file: %s", string(bodyBytes)),
				resp.StatusCode,
			)
			return
		}
		http.Redirect(w, r, PAGE_DEPLOYMENTS_LIST_PATH, http.StatusSeeOther)	
	})
}
