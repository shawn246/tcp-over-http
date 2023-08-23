package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/shawn246/tcp-over-http/scan"
	"github.com/spf13/cobra"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-quit:
			cancel()
		}
	}()

	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	//server.RegisterLogger(&logger)
	//client.RegisterLogger(&logger)
	scan.RegisterLogger(&logger)

	rootCmd := &cobra.Command{
		Use:   "toh",
		//Short: "a simple tcp tunnel transported over http",
		Short: "test tool",
	}
	//server.RegisterCmd(rootCmd)
	//client.RegisterCmd(rootCmd)
	scan.RegisterCmd(rootCmd)

	_ = rootCmd.ExecuteContext(ctx)
}
