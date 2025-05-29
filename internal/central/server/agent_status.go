package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	pba "github.com/Coosis/go-k8s-cord/internal/pb/agent/v1"
	log "github.com/sirupsen/logrus"
)

func(s *CentralServer) CheckStatus(w http.ResponseWriter, r *http.Request) {
	for agent_id, agent := range s.agents {
		log.Debugf("Checking status for agent %s", agent_id)
		resp, err := s.etcd.Get(r.Context(), agent_id)
		name := agent.Name
		if err != nil {
			w.Write(fmt.Appendf([]byte{}, "%v: %v\n", name, "offline"))
			continue
		}
		var mx int64
		mx = 0
		for _, kv := range resp.Kvs {
			cval, err := strconv.ParseInt(string(kv.Value), 10, 64)
			if err != nil {
				w.Write(fmt.Appendf([]byte{}, "%v: %v\n", name, "offline"))
				continue
			}
			mx = max(mx, cval)
		}

		valid := time.Now().Add(time.Second * -time.Duration(s.Config.AliveInterval)).Unix()
		if mx < valid {
			w.Write(fmt.Appendf([]byte{}, "%v: %v\n", name, "offline"))
		} else {
			w.Write(fmt.Appendf([]byte{}, "%v: %v\n", name, "online"))
		}
	}
}

func(s *CentralServer) CheckPods(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var buf []byte
	for agent_id, agent := range s.agents {
		log.Debugf("Pods for agent %s", agent_id)
		name := agent.Name

		client := pba.NewAgentServiceClient(agent.AgentConn)
		resp, err := client.ListPods(ctx, &pba.ListPodsRequest{ })
		if err != nil {
			buf = append(buf, fmt.Sprintf("%s: %v\n", name, "error listing pods")...)
			log.Errorf("Failed to list pods for agent %s: %v", name, err)
			continue
		}

		pods := resp.GetPods()
		for _, pod := range pods {
			buf = append(buf, fmt.Sprintf("%s: %s/%s\n", name, pod.GetNamespace(), pod.GetName())...)
		}
	}

	if len(buf) == 0 {
		buf = append(buf, "No pods found\n"...)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write(buf)
	log.Debugf("Pods check completed, %d bytes sent", len(buf))
}

