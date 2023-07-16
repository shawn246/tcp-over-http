package client

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var log *zerolog.Logger

type config struct {
	Port   int
	Url    string
	Target string
}

func RegisterLogger(logger *zerolog.Logger) {
	log = logger
}

func RegisterCmd(root *cobra.Command) {
	cfg := &config{}
	clientCmd := &cobra.Command{
		Use:   "client",
		Short: "proxy client mode",
		Run: func(cmd *cobra.Command, args []string) {
			run(cmd.Context(), cfg)
		},
	}
	clientCmd.Flags().IntVarP(&cfg.Port, "port", "p", 9090, "listen port")
	clientCmd.Flags().StringVarP(&cfg.Url, "url", "u", "", "proxy server url, example: http://localhost:9000/proxy")
	clientCmd.Flags().StringVarP(&cfg.Target, "target", "t", "mysql:3306", "target network, example: mysql:3306")
	//_ = clientCmd.MarkFlagRequired("url")
	//_ = clientCmd.MarkFlagRequired("target")

	root.AddCommand(clientCmd)
}

func run(ctx context.Context, cfg *config) {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		log.Error().Err(err).Msgf("listen failed")
		return
	}
	log.Info().Msgf("proxy client listen at :%d", cfg.Port)

	go func() {
		select {
		case <-ctx.Done():
			_ = l.Close()
		}
	}()

	for {
		conn, err := l.Accept()
		if errors.Is(err, net.ErrClosed) {
			break
		} else if err != nil {
			log.Error().Err(err).Msgf("accept failed")
			continue
		}

		g := newAgent(conn, cfg.Url, cfg.Target)
		go g.work()
	}

	log.Info().Msgf("proxy client shutdown")
}
