package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	. "github.com/Coosis/go-k8s-cord/internal"
	. "github.com/Coosis/go-k8s-cord/internal/central/model"

	// pba "github.com/Coosis/go-k8s-cord/internal/pb/agent/v1"
	pbc "github.com/Coosis/go-k8s-cord/internal/pb/central/v1"
)

type CentralServer struct {
	s *http.Server
	gs *grpc.Server
	// cp *x509.CertPool
	// Client *http.Client
	etcd *clientv3.Client

	tlsConfig *tls.Config
	Config *CentralConfig

	// map of agent id to agent name
	// registeredAgents map[string]string
	// map of agent id to agent gRPC endpoint
	// AgentGrpcEndpoints map[string]string
	agents map[string]*AgentMetadata

	pbc.UnimplementedCentralServiceServer
}

func NewCentralServer(
) (*CentralServer, error) {
	cas, err := LoadCAPool("ca.crt")
	if err != nil {
		log.Error("Failed to load CA certificate pool:", err)
		return nil, err
	}

	mux := http.NewServeMux()

	// generating central config if not present
	_, err = os.Stat(CENTRAL_CONFIG)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info("Central config file not found, creating a new one...")
			file, err := os.Create(CENTRAL_CONFIG)
			if err != nil {
				log.Error("Failed to create central config file:", err)
				return nil, err
			}
			defer file.Close()
			log.Info("Central config file created successfully")
		} else {
			log.Error("Failed to create central config:", err)
			return nil, err
		}
	}

	CentralServerConfigSetup()

	cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
	if err != nil {
		err := fmt.Errorf("Failed to load server certificate and key: %v", err)
		return nil, err
	}
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		ClientCAs: cas,
		RootCAs: cas,
		ClientAuth: tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{cert},
	}


	grpcCreds := credentials.NewTLS(tlsConfig)
	grpcServer := grpc.NewServer(grpc.Creds(grpcCreds))

	// etcd
	etcd_client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{DEFAULT_ETCD_ADDR},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Error("Failed to create etcd client:", err)
		return nil, err
	}

	config := NewCentralConfig()

	s := &http.Server {
		Addr: config.HTTPSPort,
		Handler: mux,
		// TLSConfig: tlsConfig,
	}

	cs := &CentralServer{
		s:  s,
		gs: grpcServer,
		// cp: cas,
		// Client: client,
		etcd: etcd_client,
		tlsConfig: tlsConfig,
		Config: config,
		// registeredAgents: make(map[string]string),
		// AgentGrpcEndpoints: make(map[string]string),
		agents: make(map[string]*AgentMetadata),
	}
	pbc.RegisterCentralServiceServer(grpcServer, cs)
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Info("Config file changed: ", e.Name)
		cs.Config = NewCentralConfig()
		log.Info("Reloaded config for central")
	})
	viper.WatchConfig()

	return cs, nil
}

func(s *CentralServer) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

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
		for id, agent := range s.agents {
			if agent.AgentConn != nil {
				log.Debugf("Closing gRPC connection for agent %s", id)
				if err := agent.AgentConn.Close(); err != nil {
					log.Error("Failed to close gRPC connection for agent ", id, ": ", err)
				} else {
					log.Debugf("gRPC connection for agent %s closed successfully", id)
				}
			} else {
				log.Debugf("No gRPC connection to close for agent %s", id)
			}
		}
		log.Info("gRPC server shutdown successfully")
	}()

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		log.Info("Starting central http server on ", s.s.Addr)
		s.HandleFunc("/status", LocalOnly(s.CheckStatus))
		s.HandleFunc("/stat", LocalOnly(func(w http.ResponseWriter, r *http.Request) {
			s.CheckPods(ctx, w, r)
		}))

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
		log.Info("Starting central gRPC server on ", s.Config.GRPCPort)
		if err := s.gs.Serve(lis); err != nil {
			return err
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		log.Error("Error in central server: ", err)
		cancel()
		return err
	}
	return nil
}

func(s *CentralServer) HandleFunc(
	pat string,
	f func(http.ResponseWriter, *http.Request),
) {
	s.s.Handler.(*http.ServeMux).HandleFunc(pat, f)
}

func CentralServerConfigSetup() error {
	// loading agent config
	viper.SetConfigName(CENTRAL_CONFIG)
	viper.AddConfigPath(".")
	viper.SetConfigType("yaml")

	viper.SetDefault("version", "1.0.0")
	viper.SetDefault("alive_interval", DEFAULT_ALIVE_INTERVAL)
	viper.SetDefault("https_port", DEFAULT_HTTPS_PORT)
	viper.SetDefault("grpc_port", DEFAULT_GRPC_PORT)
	viper.SetDefault("etcd_addr", DEFAULT_ETCD_ADDR)

	err := viper.ReadInConfig()
	if err != nil {
		log.Error("Failed to read central config: ", err)
		return err
	}
	err = viper.WriteConfig()
	if err != nil {
		log.Error("Failed to write central config: ", err)
		return err
	}
	return nil
}
