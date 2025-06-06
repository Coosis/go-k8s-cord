package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"k8s.io/client-go/kubernetes"

	. "github.com/Coosis/go-k8s-cord/internal"
	. "github.com/Coosis/go-k8s-cord/internal/agent/cluster"
	. "github.com/Coosis/go-k8s-cord/internal/agent/model"

	log "github.com/sirupsen/logrus"

	pba "github.com/Coosis/go-k8s-cord/internal/pb/agent/v1"
)

type AgentServer struct {
	s      *http.Server
	gs    *grpc.Server

	centralConn   *grpc.ClientConn

	k8sClientSet *kubernetes.Clientset

	tlsConfig *tls.Config

	pba.UnimplementedAgentServiceServer
}

// Does these:
// - loading config + hook up viper to handle config changes at runtime
// - CA pool loading + agent certificate loading -> tls config construction
func NewAgentServer() (*AgentServer, error) {
	// generating agent config if not present
	_, err := os.Stat(AGENT_CONFIG)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("Agent config file not found, creating a new one...")
			file, err := os.Create(AGENT_CONFIG)
			if err != nil {
				log.Error("Failed to create agent config file: ", err)
				return nil, err
			}
			defer file.Close()
		} else {
			return nil, fmt.Errorf("Failed to create agent config: %v", err)
		}
	}

	// loading agent config
	cfg := GetAgentConfig()

	// loading CA certificate pool
	cas, err := LoadCAPool("ca.crt")
	if err != nil {
		log.Error("Failed to load CA certificate pool: ", err)
		return nil, err
	}

	cert, err := tls.LoadX509KeyPair("agent.crt", "agent.key")
	if err != nil {
		return nil, fmt.Errorf("Failed to load agent certificate and key: %v", err)
	}
	// tls config construction
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		ClientCAs: cas,
		RootCAs: cas,
		ClientAuth: tls.RequestClientCert,
		Certificates: []tls.Certificate{cert},
	}

	// http server bootstrapping
	mux := http.NewServeMux()
	s := http.Server{
		Addr:    cfg.HTTPSPort,
		Handler: mux,
		TLSConfig: tlsConfig,
	}

	if cfg.Registered {
		log.Debugf("Agent is already registered, heartbeat is going to start "+
			"with an interval of %d seconds",
			cfg.HeartbeatInterval)
	}

	// gRPC server bootstrapping
	cred := credentials.NewTLS(tlsConfig)
	conn, err := grpc.NewClient(cfg.GrpcCentral, grpc.WithTransportCredentials(cred))
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to gRPC server: %v", err)
	}
	grpcServer := grpc.NewServer(grpc.Creds(cred))

	clientSet, err := GetClientSet()
	if err != nil {
		return nil, fmt.Errorf("Failed to get Kubernetes client set: %v", err)
	}

	agentServer := &AgentServer{
		s:      &s,
		gs:     grpcServer,
		centralConn:   conn,
		k8sClientSet: clientSet,

		tlsConfig: tlsConfig,
	}
	pba.RegisterAgentServiceServer(agentServer.gs, agentServer)

return agentServer, nil
}

func (s *AgentServer) HandleFunc(
	pat string,
	f func(http.ResponseWriter, *http.Request),
) {
	s.s.Handler.(*http.ServeMux).HandleFunc(pat, f)
}

func (s *AgentServer) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cfg := GetAgentConfig()

	s.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		msg, err := Healthz(ctx, *s.k8sClientSet)
		if err != nil {
			log.Error("Healthz check failed: ", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Write([]byte(msg))
	})
	s.HandleFunc("/livez", func(w http.ResponseWriter, r *http.Request) {
		msg, err := Livez(ctx, *s.k8sClientSet)
		if err != nil {
			log.Error("Livez check failed: ", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Write([]byte(msg))
	})
	s.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		msg, err := Readyz(ctx, *s.k8sClientSet)
		if err != nil {
			log.Error("Readyz check failed: ", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Write([]byte(msg))
	})
	s.HandleFunc("/pods/list", func(w http.ResponseWriter, r *http.Request) {
		pods, err := GetPods(ctx, s.k8sClientSet, "default")
		if err != nil {
			log.Error("Failed to list pods: ", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(pods); err != nil {
			log.Error("Failed to encode pods: ", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})
	if !cfg.Registered {
		log.Info("Agent not registered, registering...")
		id, err := s.Register(ctx)
		if err != nil {
			return fmt.Errorf("Failed to register agent: %w", err)
		}
		viper.Set("registered", true)
		viper.Set("uuid", id)
		WriteBackAndReload()
		cfg = GetAgentConfig()
	}

	go func() {
		<- ctx.Done()
		log.Debug("Context cancelled, shutting down servers...")

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := s.s.Shutdown(shutdownCtx); err != nil {
			log.Error("Failed to shutdown http server: ", err)
		} else {
			log.Info("Http server shutdown successfully")
		}

		s.gs.GracefulStop()
		s.centralConn.Close()
		log.Info("gRPC server shutdown successfully")
	}()

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		log.Info("Starting central http server on ", s.s.Addr)
		// if err := s.s.ListenAndServeTLS("server.crt", "server.key"); err != nil {
		if err := s.s.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				return err
			}
		}
		return nil
	})
	g.Go(func() error {
		lis, err := net.Listen("tcp", cfg.GRPCPort)
		if err != nil {
			return fmt.Errorf("failed to listen: %v", err)
		}
		log.Info("Starting agent gRPC server on ", cfg.GRPCPort)
		if err := s.gs.Serve(lis); err != nil {
			return err
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		cancel()
		return fmt.Errorf("error in agent server: %w", err)
	}
	return nil
}
