package main

import (
	"context"
	"os"
	"os/signal"

	. "github.com/Coosis/go-k8s-cord/internal/central/server"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use: "serve",
	Short: "start the central control server",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := NewCentralServer()
		if err != nil {
			log.Error("Failed to create central server:", err)
			return err
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
		defer stop()
		g, ctx := errgroup.WithContext(ctx)

		g.Go(func() error {
			return s.Start(ctx)
		})

		if err := g.Wait(); err != nil {
			log.Fatal("Server shutdown due to error:", err)
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

