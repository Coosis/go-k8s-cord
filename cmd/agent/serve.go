package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	. "github.com/Coosis/go-k8s-cord/internal/agent/server"
	. "github.com/Coosis/go-k8s-cord/internal/agent/model"

	log "github.com/sirupsen/logrus"
)

var serveCmd = &cobra.Command{
	Use: "serve",
	Short: "start the agent server",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := NewAgentServer()
		if err != nil {
			return err
		}

		cfg := GetAgentConfig()

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
		defer stop()

		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			return s.Start(ctx)
		})

		g.Go(func() error {
			ticker := time.NewTicker(time.Second * time.Duration(cfg.HeartbeatInterval))
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					log.Debug("Sending heartbeat...")
					err := s.Heartbeat(ctx)
					if err != nil {
						log.Error("Failed to send heartbeat:", err)
					} else {
						log.Debug("Heartbeat sent successfully")
					}
				case <-ctx.Done():
					log.Debug("Stopping heartbeat...")
					return nil
				}
			}
		})


		// cronjobs for server
		// ?

		if err := g.Wait(); err != nil {
			log.Fatal("shutdown due to error:", err)
		}

		log.Info("Shutting down server...")

		return nil
	},
}

func init() {
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stdout)
	rootCmd.AddCommand(serveCmd)
}
