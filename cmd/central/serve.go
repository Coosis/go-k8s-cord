package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	. "github.com/Coosis/go-k8s-cord/internal/central/server"

	log "github.com/sirupsen/logrus"
)

var (
	WEB_ENDPOINT = "http://localhost%s"
)

var serveCmd = &cobra.Command{
	Use: "serve",
	Short: "start the central control server",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := NewCentralServer()
		if err != nil {
			return err
		}

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
		defer stop()
		g, ctx := errgroup.WithContext(ctx)

		g.Go(func() error {
			return s.Start(ctx)
		})

		if err := g.Wait(); err != nil {
			return fmt.Errorf("server error: %w", err)
		}

		return nil
	},
}

func init() {
	log.SetLevel(log.DebugLevel)
	log.SetOutput(os.Stdout)
	rootCmd.AddCommand(serveCmd)
}

