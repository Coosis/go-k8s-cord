// per-agent api poor implementation, look for "agent_page.go" for 
// htmx integration and html templates.
package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	mux "github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	. "github.com/Coosis/go-k8s-cord/internal/central/model"

	pba "github.com/Coosis/go-k8s-cord/internal/pb/agent/v1"
)

const (
	AGENT_STATUS             = "/api/v1/agent/{agent_id}/status"
	AGENT_DEPLOYMENTS        = "/api/v1/agent/{agent_id}/deployments"
	AGENT_DEPLOYMENTS_APPLY  = "/api/v1/agent/{agent_id}/deployments/apply"
	AGENT_DEPLOYMENTS_REMOVE = "/api/v1/agent/{agent_id}/deployments/remove"
	AGENT_DEPLOYMENTS_HASH   = "/api/v1/agent/{agent_id}/deployments/hash"
	AGENT_LIST               = "/api/v1/agent"
)

func (s *CentralServer) setupAgentRoutes() {
	s.HandleFunc(AGENT_STATUS, s.agentStatus)
	s.HandleFunc(AGENT_DEPLOYMENTS, s.agentDeployments)
	s.HandleFunc(AGENT_DEPLOYMENTS_APPLY, s.agentApplyDeployments)
	s.HandleFunc(AGENT_DEPLOYMENTS_REMOVE, s.agentRemoveDeployments)
	s.HandleFunc(AGENT_DEPLOYMENTS_HASH, s.agentDeploymentsHash)
	s.HandleFunc(AGENT_LIST, s.listAgents)
}

func (s *CentralServer) agentStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Warn("Method not allowed for agent status endpoint")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	log.Debug("Handling agent status request")

	cfg := GetCentralConfig()

	vars := mux.Vars(r)
	agentID := vars["agent_id"]
	resp, err := s.etcd.Get(r.Context(), agentID)
	if err != nil {
		log.Errorf("Failed to get agent status for %s: %v", agentID, err)
		http.Error(w, "Failed to get agent status", http.StatusInternalServerError)
		return
	}
	agent := s.agents[agentID]
	onlineOrNot := "offline"
	var mx int64
	mx = 0
	for _, kv := range resp.Kvs {
		cval, err := strconv.ParseInt(string(kv.Value), 10, 64)
		if err != nil {
			log.Errorf("Failed to parse agent status for %s: %v", agentID, err)
			http.Error(w, "Failed to parse agent status", http.StatusInternalServerError)
			return
		}
		mx = max(mx, cval)
	}
	valid := time.Now().Add(time.Second * -time.Duration(cfg.AliveInterval)).Unix()
	if mx < valid {
		onlineOrNot = "offline"
	} else {
		onlineOrNot = "online"
	}

	name := agent.Name
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"name":   name,
		"status": onlineOrNot,
	}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Errorf("Failed to encode agent status response for %s: %v", name, err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	log.Debugf("Agent %s status: %s", name, onlineOrNot)
}

func (s *CentralServer) agentDeployments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Warn("Method not allowed for agent deployments endpoint")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	log.Debug("Handling list deployments request")

	vars := mux.Vars(r)
	agentID := vars["agent_id"]
	client := pba.NewAgentServiceClient(s.agents[agentID].AgentConn)
	resp, err := client.ListDeployments(r.Context(), &pba.ListDeploymentsRequest{})
	if err != nil {
		log.Errorf("Failed to list deployments for agent %s: %v", agentID, err)
		http.Error(w, "Failed to list deployments: "+err.Error(), http.StatusInternalServerError)
		return
	}

	metadata := []map[string]any{}
	for _, deployment := range resp.GetDeployments() {
		metadata = append(metadata, map[string]any{
			"apiVersion":        deployment.GetApiVersion(),
			"name":              deployment.GetName(),
			"uid":               deployment.GetUid(),
			"availableReplicas": deployment.GetAvailableReplicas(),
			"replicas":          deployment.GetReplicas(),
			"readyReplicas":     deployment.GetReadyReplicas(),
			"creationTimestamp": deployment.GetCreationTimestamp(),
			"updatedReplicas":   deployment.GetUpdatedReplicas(),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(metadata)
	if err != nil {
		log.Errorf("Failed to encode deployments for agent %s: %v", agentID, err)
		http.Error(w, "Failed to encode deployments: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *CentralServer) listAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Warn("Method not allowed for list agents endpoint")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var agents []map[string]string
	for agentID, agent := range s.agents {
		agents = append(agents, map[string]string{
			"id":   agentID,
			"name": agent.Name,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(agents)
	if err != nil {
		log.Errorf("Failed to encode agents: %v", err)
		http.Error(w, "Failed to encode agents: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

type applyDeploymentsPayload struct {
	DeploymentFiles []string `json:"deployment_files"`
}

func (s *CentralServer) agentApplyDeployments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Warn("Method not allowed for agent apply deployments endpoint")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	log.Debug("Handling apply deployments request")

	vars := mux.Vars(r)
	agentID := vars["agent_id"]
	client := pba.NewAgentServiceClient(s.agents[agentID].AgentConn)

	var payload applyDeploymentsPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		log.Errorf("Failed to decode deployment files for agent %s: %v", agentID, err)
		http.Error(w, "Failed to decode deployment files: "+err.Error(), http.StatusBadRequest)
		return
	}

	_, err = client.ApplyDeployments(r.Context(), &pba.ApplyDeploymentsRequest{
		DeploymentName: payload.DeploymentFiles,
	})
	if err != nil {
		log.Errorf("Failed to apply deployments for agent %s: %v", agentID, err)
		http.Error(w, "Failed to apply deployments: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Deployments applied successfully"))
}

type removeDeploymentsPayload struct {
	DeploymentFiles []string `json:"deployment_files"`
}

func (s *CentralServer) agentRemoveDeployments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Warn("Method not allowed for agent remove deployments endpoint")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	log.Debug("Handling remove deployments request")

	vars := mux.Vars(r)
	agentID := vars["agent_id"]
	client := pba.NewAgentServiceClient(s.agents[agentID].AgentConn)

	var payload removeDeploymentsPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		log.Errorf("Failed to decode deployment files for agent %s: %v", agentID, err)
		http.Error(w, "Failed to decode deployment files: "+err.Error(), http.StatusBadRequest)
		return
	}

	_, err = client.RemoveDeployments(r.Context(), &pba.RemoveDeploymentsRequest{
		DeploymentName: payload.DeploymentFiles,
	})
	if err != nil {
		log.Errorf("Failed to remove deployments for agent %s: %v", agentID, err)
		http.Error(w, "Failed to remove deployments: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Deployments removed successfully"))
}

func(s *CentralServer) agentDeploymentsHash(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		log.Warn("Method not allowed for agent deployments hash endpoint")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	log.Debug("Handling deployments hash request")

	vars := mux.Vars(r)
	agentID := vars["agent_id"]
	client := pba.NewAgentServiceClient(s.agents[agentID].AgentConn)

	resp, err := client.GetDeploymentsHash(r.Context(), &pba.GetDeploymentsHashRequest{})
	if err != nil {
		log.Errorf("Failed to get deployments hash for agent %s: %v", agentID, err)
		http.Error(w, "Failed to get deployments hash: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if resp == nil || *resp.Hash == "" {
		log.Warnf("No deployments hash found for agent %s", agentID)
		http.Error(w, "No deployments hash found for agent", http.StatusNotFound)
		return
	}

	hash := make(map[string]string)
	hash["hash"] = *resp.Hash

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(hash)
	if err != nil {
		log.Errorf("Failed to encode deployments hash for agent %s: %v", agentID, err)
		http.Error(w, "Failed to encode deployments hash: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
