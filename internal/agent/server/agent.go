package server

import (
	"context"
	"crypto/tls"

	// "crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	yaml "github.com/go-yaml/yaml"
	"k8s.io/client-go/kubernetes"

	// "github.com/gogo/protobuf/proto"
	log "github.com/sirupsen/logrus"
	viper "github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	. "github.com/Coosis/go-k8s-cord/internal"
	. "github.com/Coosis/go-k8s-cord/internal/agent/cluster"
	. "github.com/Coosis/go-k8s-cord/internal/agent/model"

	pba "github.com/Coosis/go-k8s-cord/internal/pb/agent/v1"
	// pbc "github.com/Coosis/go-k8s-cord/pkg/central/v1"
)

const (
	ADDR_PLACEHOLDER = "CHANGE_ME_TO_AGENT_ENDPOINT"
	EDIT_CONFIG_MESSAAGE = "Please edit the config file " +
	"before re-starting the agent. Make sure to set the agent's address so " +
	"that the central server can communicate with it. Make sure it matches the " +
	"field in the certificate."
	AGENT_CONFIG_NOT_SET = "Agent address is not set in the config file. " +
	"Please edit the config file and set the address before continuing."
)

type AgentServer struct {
	s      *http.Server
	gs    *grpc.Server

	centralConn   *grpc.ClientConn

	k8sClientSet *kubernetes.Clientset

	tlsConfig *tls.Config
	Config *AgentConfig

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
			// log.Info("Agent config file created successfully")
			// return nil, fmt.Errorf("%v", EDIT_CONFIG_MESSAAGE)
		} else {
			return nil, fmt.Errorf("Failed to create agent config: %v", err)
		}
	}

	// loading agent config
	err = AgentServerConfigSetup()
	if err != nil {
		return nil, fmt.Errorf("Failed to setup agent config: %v", err)
	}
	config := NewAgentConfig()
	if config.Addr == ADDR_PLACEHOLDER {
		return nil, fmt.Errorf("%v", AGENT_CONFIG_NOT_SET)
	}
	// ---------------------------------------------------------------

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
		Addr:    config.HTTPSPort,
		Handler: mux,
		TLSConfig: tlsConfig,
	}

	if config.Registered {
		log.Debugf("Agent is already registered, heartbeat is going to start "+
			"with an interval of %d seconds",
			config.HeartbeatInterval)
	}

	// gRPC server bootstrapping
	cred := credentials.NewTLS(tlsConfig)
	conn, err := grpc.NewClient(config.GrpcCentral, grpc.WithTransportCredentials(cred))
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
		Config: config,
	}
	pba.RegisterAgentServiceServer(agentServer.gs, agentServer)

	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Info("Config file changed: ", e.Name)
		agentServer.Config = NewAgentConfig()
		log.Info("Reloaded config for agent: ", config.AgentName)
	})
	viper.WatchConfig()

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
	if !s.Config.Registered {
		log.Info("Agent not registered, registering...")
		s.Register(ctx)
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
		lis, err := net.Listen("tcp", s.Config.GRPCPort)
		if err != nil {
			return fmt.Errorf("failed to listen: %v", err)
		}
		log.Info("Starting agent gRPC server on ", s.Config.GRPCPort)
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

func AgentServerConfigSetup() error {
	// loading agent config
	viper.SetConfigName(AGENT_CONFIG)
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")

	viper.SetDefault("agent_name", "default_agent")
	viper.SetDefault("addr", ADDR_PLACEHOLDER)
	viper.SetDefault("version", "1.0.0")
	viper.SetDefault("grpc_central", DEFAULT_GRPC_CENTRAL)
	viper.SetDefault("grpc_port", DEFAULT_GRPC_PORT)
	viper.SetDefault("https_central", DEFAULT_HTTPS_CENTRAL)
	viper.SetDefault("https_port", DEFAULT_HTTPS_PORT)
	viper.SetDefault("uuid", "")
	viper.SetDefault("registered", false)
	viper.SetDefault("heartbeat_interval", DEFAULT_HEARTBEAT_INTERVAL)

	err := viper.ReadInConfig()
	if err != nil {
		log.Error("Failed to read agent config: ", err)
		return err
	}
	err = viper.WriteConfig()
	if err != nil {
		log.Error("Failed to write agent config: ", err)
		return err
	}
	return nil
}

func (s *AgentServer) WriteConfig() error {
	file, err := os.OpenFile(AGENT_CONFIG, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Error("Failed to open agent config file: ", err)
		return err
	}
	defer file.Close()

	enc := yaml.NewEncoder(file)
	if err := enc.Encode(s.Config); err != nil {
		log.Error("Failed to encode agent config: ", err)
		return err
	}

	log.Info("Agent config file updated successfully")
	return nil
}
