package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	log "github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
	gogit "github.com/go-git/go-git/v5"
	mux "github.com/gorilla/mux"

	. "github.com/Coosis/go-k8s-cord/internal"
	. "github.com/Coosis/go-k8s-cord/internal/central/model"

	pbc "github.com/Coosis/go-k8s-cord/internal/pb/central/v1"
)

type CentralServer struct {
	s *http.Server
	gs *grpc.Server
	etcd *clientv3.Client

	tlsConfig *tls.Config

	agents map[string]*AgentMetadata

	repo *gogit.Repository

	pbc.UnimplementedCentralServiceServer
}

func NewCentralServer(
) (*CentralServer, error) {
	cas, err := LoadCAPool("ca.crt")
	if err != nil {
		log.Error("Failed to load CA certificate pool:", err)
		return nil, err
	}

	mux := mux.NewRouter()

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

	cfg := GetCentralConfig()

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
		Endpoints:   []string{cfg.EtcdAddr},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Error("Failed to create etcd client:", err)
		return nil, err
	}

	s := &http.Server {
		Addr: cfg.HTTPSPort,
		Handler: mux,
	}

	cs := &CentralServer{
		s:  s,
		gs: grpcServer,
		etcd: etcd_client,
		tlsConfig: tlsConfig,
		agents: make(map[string]*AgentMetadata),
	}
	cs.GitInit()
	pbc.RegisterCentralServiceServer(grpcServer, cs)

	return cs, nil
}

func(s *CentralServer) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	cfg := GetCentralConfig()

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

		// static dir
		s.s.Handler.(*mux.Router).
		PathPrefix("/static/").
		Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

		s.setupDeploymentRoutes()
		s.setupStatusRoutes()
		s.setupAgentRoutes()

		s.setupRootHTML()
		s.setupStatusHTML()
		s.setupAgentsHTML()
		s.setupDeploymentsHTML()

		log.Infof("Visit http://localhost%s to access the central ui", s.s.Addr)

		// Swap this out if you wish to use tls
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
		log.Info("Starting central gRPC server on ", cfg.GRPCPort)
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

func(s *CentralServer) GitInit() error {
	cfg := GetCentralConfig()
	repo, err := gogit.PlainOpen(cfg.DeploymentsDir)
	if err != nil {
		return fmt.Errorf("Failed to open repository: %v", err)
	}

	s.repo = repo

	// Check if the remote already exists
	_, err = repo.Remotes()
	if err != nil {
		return fmt.Errorf("Failed to get remotes: %v, maybe remote has not been set", err)
	}

	return nil
}

func(s *CentralServer) HandleFunc(
	pat string,
	f func(http.ResponseWriter, *http.Request),
) {
	s.s.Handler.(*mux.Router).HandleFunc(pat, f)
}
