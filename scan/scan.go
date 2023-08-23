package scan

import (
	"context"
	"net"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var log *zerolog.Logger

type config struct {
	Host string
}

func RegisterLogger(logger *zerolog.Logger) {
	log = logger
}

func RegisterCmd(root *cobra.Command) {
	cfg := &config{}
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "scan specified port",
		Run: func(cmd *cobra.Command, args []string) {
			run(cmd.Context(), cfg)
		},
	}
	scanCmd.Flags().StringVarP(&cfg.Host, "host", "h", "", "remote address, example 192.168.10.10:502")
	_ = scanCmd.MarkFlagRequired("host")
	root.AddCommand(scanCmd)
}

func run(ctx context.Context, cfg *config) {
	conn, err := net.DialTimeout("tcp", cfg.Host, time.Second*5)
	if err != nil {
		log.Error().Err(err).Msgf("failed to connect %s", cfg.Host)
		return
	}

	defer conn.Close()
	log.Info().Msgf("success to connect %s", cfg.Host)
}