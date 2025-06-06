// per-agent htmx integration, check agent_api.go for the API implementation.
package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	. "github.com/Coosis/go-k8s-cord/internal/central/deployment"
	. "github.com/Coosis/go-k8s-cord/internal/central/model"

	mux "github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

const (
	agentStatusEndpoint            = "http://localhost%s/api/v1/agent/%s/status"
	agentDeploymentsEndpoint       = "http://localhost%s/api/v1/agent/%s/deployments"
	agentDeploymentsHashEndpoint   = "http://localhost%s/api/v1/agent/%s/deployments/hash"
	agentApplyDeploymentsEndpoint  = "http://localhost%s/api/v1/agent/%s/deployments/apply"
	agentRemoveDeploymentsEndpoint = "http://localhost%s/api/v1/agent/%s/deployments/remove"
	agentListEndpoint              = "http://localhost%s/api/v1/agent"
)

func (s *CentralServer) setupAgentsHTML() {
	cfg := GetCentralConfig()

	// templ := template.Must(template.ParseFiles("templates/base.html"))
	agentCardTempl := template.Must(template.ParseFiles(
		"templates/base.html",
		"templates/agent_card.html",
	))
	s.HandleFunc("/agent", func(w http.ResponseWriter, r *http.Request) {
		var agents []map[string]string
		resp, err := http.Get(fmt.Sprintf(agentListEndpoint, cfg.HTTPSPort))
		if err != nil {
			http.Error(w, "Failed to get agent list: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Error("Failed to get agent list, status code: ", resp.StatusCode)
			http.Error(w, "Failed to get agent list, status code: "+resp.Status, resp.StatusCode)
			return
		}
		if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
			log.Error("Failed to decode agent list: ", err)
			http.Error(w, "Failed to decode agent list: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		for _, agent := range agents {
			id := agent["id"]
			resp, err := http.Get(fmt.Sprintf(agentStatusEndpoint, cfg.HTTPSPort, id))
			if err != nil {
				log.Error("Failed to get agent status: ", err)
				agent["status"] = "Error..."
				continue
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				log.Error("Failed to get agent status, status code: ", resp.StatusCode)
				agent["status"] = "Error..."
				continue
			}

			agentStatus := make(map[string]string)
			if err := json.NewDecoder(resp.Body).Decode(&agentStatus); err != nil {
				log.Error("Failed to decode agent status: ", err)
				agent["status"] = "Error..."
				continue
			}
			status, ok := agentStatus["status"]
			if !ok {
				log.Error("Agent status not found in status response")
				agent["status"] = "Error..."
				continue
			}

			agent["status"] = status
		}
		htmlVars := map[string]any{
			"Agents": agents,
		}
		if err := agentCardTempl.Execute(w, htmlVars); err != nil {
			log.Error("Failed to execute agent card template: ", err)
			http.Error(w, "Failed to execute agent template: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})

	agentTempl := template.Must(template.ParseFiles(
		"templates/base.html",
		"templates/agent.html",
	))
	s.HandleFunc("/agent/{agent_id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		agentID := vars["agent_id"]

		// GET to agent status
		endpoint := fmt.Sprintf(agentStatusEndpoint, cfg.HTTPSPort, agentID)
		resp, err := http.Get(endpoint)
		if err != nil {
			log.Error("Failed to get agent status: ", err)
			http.Error(w, "Failed to get agent status: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Error("Failed to get agent status, status code: ", resp.StatusCode)
			http.Error(w, "Failed to get agent status, status code: "+resp.Status, resp.StatusCode)
			return
		}
		agentStatus := make(map[string]string)
		if err := json.NewDecoder(resp.Body).Decode(&agentStatus); err != nil {
			log.Error("Failed to decode agent status: ", err)
			http.Error(w, "Failed to decode agent status: "+err.Error(), http.StatusInternalServerError)
			return
		}

		agentName, ok := agentStatus["name"]
		if !ok {
			log.Error("Agent name not found in status response")
			http.Error(w, "Agent name not found in status response", http.StatusInternalServerError)
			return
		}

		agentStatusValue, ok := agentStatus["status"]
		if !ok {
			log.Error("Agent status not found in status response")
			http.Error(w, "Agent status not found in status response", http.StatusInternalServerError)
			return
		}

		log.Debugf("Agent %s status: %s", agentName, agentStatusValue)

		// GET to agent deployments
		var deployments []map[string]any
		resp, err = http.Get(fmt.Sprintf(agentDeploymentsEndpoint, cfg.HTTPSPort, agentID))
		if err != nil {
			log.Error("Failed to get agent deployments: ", err)
			http.Error(w, "Failed to get agent deployments: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Error("Failed to get agent deployments, status code: ", resp.StatusCode)
			http.Error(w, "Failed to get agent deployments, status code: "+resp.Status, resp.StatusCode)
			return
		}
		if err := json.NewDecoder(resp.Body).Decode(&deployments); err != nil {
			http.Error(w, "Failed to decode agent deployments: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// GET to agent deployments hash
		resp, err = http.Get(fmt.Sprintf(agentDeploymentsHashEndpoint, cfg.HTTPSPort, agentID))
		if err != nil {
			log.Error("Failed to get agent deployments hash: ", err)
			http.Error(w, "Failed to get agent deployments hash: "+err.Error(), http.StatusInternalServerError)
			return
		}
		hash := make(map[string]string)
		if resp.StatusCode != http.StatusOK {
			log.Error("Failed to get agent deployments hash, status code: ", resp.StatusCode)
			http.Error(w, "Failed to get agent deployments hash, status code: "+resp.Status, resp.StatusCode)
			return
		}
		defer resp.Body.Close()
		if err := json.NewDecoder(resp.Body).Decode(&hash); err != nil {
			log.Error("Failed to decode agent deployments hash: ", err)
			http.Error(w, "Failed to decode agent deployments hash: "+err.Error(), http.StatusInternalServerError)
			return
		}
		deploymentsHash, ok := hash["hash"]
		if !ok {
			log.Error("Deployments hash not found in response")
			http.Error(w, "Deployments hash not found in response", http.StatusInternalServerError)
			return
		}
		centralHash, err := DeploymentsHash(s.repo)
		if err != nil {
			log.Error("Failed to get central deployments hash: ", err)
			http.Error(w, "Failed to get central deployments hash: "+err.Error(), http.StatusInternalServerError)
			return
		}
		var hashMatch string
		if deploymentsHash == centralHash {
			hashMatch = "Sync"
		} else {
			hashMatch = "Out of Sync, please wait..."
		}

		// GET to get all available deployment files
		endpoint = fmt.Sprintf(DEPLOYMENTS_LIST_URL, cfg.HTTPSPort)
		resp, err = http.Get(endpoint)
		if err != nil {
			http.Error(w, "Failed to list deployments: "+err.Error(), http.StatusInternalServerError)
			return
		}
		// decode from json
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			http.Error(w, "Failed to list deployments, status code: "+resp.Status, resp.StatusCode)
			return
		}
		var items []string
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&items)
		if err != nil {
			http.Error(w, "Failed to decode deployments response: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		htmlVars := map[string]any{
			"AgentTitle":      fmt.Sprintf("%s - %s", agentName, agentStatusValue),
			"AgentID":         agentID,
			"Deployments":     deployments,
			"HashMatch":       hashMatch,
			"DeploymentFiles": items,
		}
		if err := agentTempl.Execute(w, htmlVars); err != nil {
			log.Error("Failed to execute agent template: ", err)
			http.Error(w, "Failed to execute agent template: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})

	agentDeploymentsTempl := template.Must(template.ParseFiles("templates/agent_deployments.html"))
	s.HandleFunc("/agent/{agent_id}/deployments", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		agentid := vars["agent_id"]

		// GET to agent deployments
		resp, err := http.Get(fmt.Sprintf(agentDeploymentsEndpoint, cfg.HTTPSPort, agentid))
		if err != nil {
			log.Error("Failed to get agent deployments: ", err)
			http.Error(w, "Failed to get agent deployments: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Error("Failed to get agent deployments, status code: ", resp.StatusCode)
			http.Error(w, "Failed to get agent deployments, status code: "+resp.Status, resp.StatusCode)
			return
		}
		var deployments []map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&deployments); err != nil {
			log.Error("Failed to decode agent deployments: ", err)
			http.Error(w, "Failed to decode agent deployments: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		htmlVars := map[string]any{
			"AgentID":     agentid,
			"Deployments": deployments,
		}
		if err := agentDeploymentsTempl.Execute(w, htmlVars); err != nil {
			log.Error("Failed to execute agent deployments template: ", err)
			http.Error(w, "Failed to execute agent deployments template: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// this exists here instead of relying solely on /api
	// is because htmx doesn't play well with sending json with body
	s.HandleFunc("/agent/{agent_id}/deployments/apply", func(w http.ResponseWriter, r *http.Request) {
		// Post because htmx doesn't use hx-vals for delete requests
		if r.Method != http.MethodPost {
			log.Warn("Method not allowed for agent deployments apply endpoint")
			http.Error(w, "Method not allwed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			log.Error("Failed to parse form data: ", err)
			http.Error(w, "Failed to parse form data: "+err.Error(), http.StatusBadRequest)
			return
		}
		agentid, ok := mux.Vars(r)["agent_id"]
		if !ok {
			log.Error("Agent ID not found in request")
			http.Error(w, "Agent ID not found", http.StatusBadRequest)
			return
		}
		deploymentFiles := r.Form["deployment_files"]
		if len(deploymentFiles) == 0 {
			log.Warn("No deployment files provided for application")
			http.Error(w, "No deployment files provided", http.StatusBadRequest)
			return
		}

		for _, file := range deploymentFiles {
			log.Debugf("Applying deployment file: %s", file)
		}

		payload := applyDeploymentsPayload{
			DeploymentFiles: deploymentFiles,
		}

		jsonBody, err := json.Marshal(payload)
		if err != nil {
			log.Error("Failed to marshal deployment files to JSON: ", err)
			http.Error(w, "Failed to marshal deployment files: "+err.Error(), http.StatusInternalServerError)
			return
		}

		reader := bytes.NewReader(jsonBody)
		resp, err := http.Post(
			fmt.Sprintf(agentApplyDeploymentsEndpoint, cfg.HTTPSPort, agentid),
			"application/json",
			reader,
		)
		if err != nil {
			log.Error("Failed to apply deployments: ", err)
			http.Error(w, "Failed to apply deployments: "+err.Error(), http.StatusInternalServerError)
			return
		}

		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Error("Failed to apply deployments, status code: ", resp.StatusCode)
			http.Error(w, "Failed to apply deployments, status code: "+resp.Status, resp.StatusCode)
			return
		}

		// Return new deployments page
		http.Redirect(w, r, fmt.Sprintf("/agent/%s/deployments", agentid), http.StatusSeeOther)
	})

	// this exists here instead of relying solely on /api
	// is because htmx doesn't play well with sending json with body
	s.HandleFunc("/agent/{agent_id}/deployments/remove", func(w http.ResponseWriter, r *http.Request) {
		// Post because htmx doesn't use hx-vals for delete requests
		if r.Method != http.MethodPost {
			log.Warn("Method not allowed for agent deployments remove endpoint")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			log.Error("Failed to parse form data: ", err)
			http.Error(w, "Failed to parse form data: "+err.Error(), http.StatusBadRequest)
			return
		}

		agentid, ok := mux.Vars(r)["agent_id"]
		if !ok {
			log.Error("Agent ID not found in request")
			http.Error(w, "Agent ID not found", http.StatusBadRequest)
			return
		}

		deploymentFiles := r.Form["deployment_files"]
		if len(deploymentFiles) == 0 {
			log.Warn("No deployment files provided for removal")
			http.Error(w, "No deployment files provided", http.StatusBadRequest)
			return
		}

		for _, file := range deploymentFiles {
			log.Debugf("Removing deployment file: %s", file)
		}

		payload := removeDeploymentsPayload{
			DeploymentFiles: deploymentFiles,
		}

		jsonBody, err := json.Marshal(payload)
		if err != nil {
			log.Error("Failed to marshal deployment files to JSON: ", err)
			http.Error(w, "Failed to marshal deployment files: "+err.Error(), http.StatusInternalServerError)
			return
		}

		reader := bytes.NewReader(jsonBody)

		_, err = http.Post(
			fmt.Sprintf(agentRemoveDeploymentsEndpoint, cfg.HTTPSPort, agentid),
			"application/json",
			reader,
		)

		if err != nil {
			log.Error("Failed to remove deployments: ", err)
			http.Error(w, "Failed to remove deployments: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Return new deployments page
		http.Redirect(w, r, fmt.Sprintf("/agent/%s/deployments", agentid), http.StatusSeeOther)
	})
}
