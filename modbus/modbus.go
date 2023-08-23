package modbus

import (
	"context"

	"github.com/goburrow/modbus"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var log *zerolog.Logger

type config struct {
	Host         string
	FunctionCode string
	Address      int
	Quantity     int
}

func RegisterLogger(logger *zerolog.Logger) {
	log = logger
}

func RegisterCmd(root *cobra.Command) {
	cfg := &config{}
	scanCmd := &cobra.Command{
		Use:   "modbus",
		Short: "scan specified port",
		Run: func(cmd *cobra.Command, args []string) {
			run(cmd.Context(), cfg)
		},
	}
	scanCmd.Flags().StringVarP(&cfg.Host, "host", "h", "", "plc的通讯地址与端口，例如192.168.10.10:502")
	scanCmd.Flags().StringVarP(&cfg.FunctionCode, "function", "f", "", "功能码, 例如01, 02, 03")
	scanCmd.Flags().IntVarP(&cfg.Address, "address", "a", 0, "起始地址")
	scanCmd.Flags().IntVarP(&cfg.Quantity, "quantity", "q", 0, "数量")

	_ = scanCmd.MarkFlagRequired("host")
	_ = scanCmd.MarkFlagRequired("function")
	_ = scanCmd.MarkFlagRequired("address")
	_ = scanCmd.MarkFlagRequired("quantity")
	root.AddCommand(scanCmd)
}

func run(ctx context.Context, cfg *config) {
	client := modbus.TCPClient(cfg.Host)
	if
	client.ReadCoils()
}
