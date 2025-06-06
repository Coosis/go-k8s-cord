// Handles the status htmx page and optionally relay htmx requests to the status API.
// You can look for "status_api.go" to check the status API implementation.
package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	. "github.com/Coosis/go-k8s-cord/internal/central/model"

	log "github.com/sirupsen/logrus"
)

const (
	statusEndpoint     = "http://localhost%s/api/v1/status"

	// serves:
	// - /status: the status page
	// - /status/list: a list of all agents and their status
	statusPage         = "/status"
	statusListEndpoint = "/status/list"

	// template used to render the list of statuses
	statusListItemTemplate = `
	{{ range $key, $val := . }}
		<li><strong>{{ $key }}</strong>: {{ $val }}</li>
	{{ end }}
	`
)

func (s *CentralServer) setupStatusHTML() {
	cfg := GetCentralConfig()
	statusTempl := template.Must(template.ParseFiles(
		"templates/base.html",
		"templates/status.html",
	))
	s.HandleFunc(statusPage, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		err := statusTempl.Execute(w, nil)
		if err != nil {
			log.Error("Failed to execute status template: ", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})

	listitems := template.Must(template.New("list").Parse(statusListItemTemplate))
	s.HandleFunc(statusListEndpoint, func(w http.ResponseWriter, r *http.Request) {
		endpoint := fmt.Sprintf(statusEndpoint, cfg.HTTPSPort)
		status, err := http.Get(endpoint)
		if err != nil {
			log.Error("Failed to get status: ", err)
			http.Error(w, "Failed to get status", http.StatusInternalServerError)
			return
		}

		defer status.Body.Close()
		if status.StatusCode != http.StatusOK {
			log.Error("Failed to get status, status code: ", status.StatusCode)
			http.Error(w, "Failed to get status", status.StatusCode)
			return
		}

		var statuses map[string]string
		// decode to json
		decoder := json.NewDecoder(status.Body)
		err = decoder.Decode(&statuses)
		if err != nil {
			log.Error("Failed to decode status response: ", err)
			http.Error(w, "Failed to decode status response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		err = listitems.Execute(w, statuses)
		if err != nil {
			log.Error("Failed to execute template: ", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})
}
