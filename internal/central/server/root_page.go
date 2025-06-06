// Root page for central server.
// Look for base.html and index.html in templates directory for the HTML structure.
package server

import (
	"html/template"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func(s *CentralServer) setupRootHTML() {
	// status
	tmpl := template.Must(template.ParseFiles(
		"templates/base.html",
		"templates/index.html",
	))
	s.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err := tmpl.Execute(w, nil)
		if err != nil {
			log.Error("Failed to execute template: ", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})
}
