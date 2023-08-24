package modbus

import (
	"context"
	"encoding/binary"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/goburrow/modbus"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

var l *zerolog.Logger

type config struct {
	Remote         string
	DeviceID int
	Address      string
	Offset int
}

func RegisterLogger(logger *zerolog.Logger) {
	l = logger
}

func RegisterCmd(root *cobra.Command) {
	cfg := &config{}
	modCmd := &cobra.Command{
		Use:   "modbus",
		Short: "try modbus get",
		Run: func(cmd *cobra.Command, args []string) {
			run(cmd.Context(), cfg)
		},
	}
	modCmd.Flags().StringVarP(&cfg.Remote, "remote", "r", "", "plc的通讯地址与端口，例如192.168.10.10:502")
	modCmd.Flags().IntVarP(&cfg.DeviceID, "device", "d", 1, "设备id，默认为1")
	modCmd.Flags().IntVarP(&cfg.Offset, "offset", "o", 0, "设备id，默认为1")
	modCmd.Flags().StringVarP(&cfg.Address, "address", "a", "", "地址")

	_ = modCmd.MarkFlagRequired("remote")
	_ = modCmd.MarkFlagRequired("address")

	root.AddCommand(modCmd)
}

func run(ctx context.Context, cfg *config) {
	handler  := modbus.NewTCPClientHandler(cfg.Remote)
	handler.SlaveId = byte(int8(cfg.DeviceID))
	handler.Logger = log.New(os.Stdout, "toh: ", log.LstdFlags)

	if err := handler.Connect(); err != nil {
		l.Error().Err(err).Msgf("fail to connect modbus server")
		return
	}
	defer handler.Close()

	client := modbus.NewClient(handler)

	cfg.Address = strings.ToUpper(cfg.Address)
	if !strings.HasPrefix(cfg.Address, "VW") &&
		!strings.HasPrefix(cfg.Address, "VD") {
		l.Error().Msgf("unsupported test address")
		return
	}

	addr, err := strconv.Atoi(cfg.Address[2:])
	if err != nil {
		l.Error().Err(err).Msgf("unsupported test address")
		return
	}

	switch cfg.Address[:2] {
	case "VW":
		res, err := client.ReadHoldingRegisters(uint16(addr/2-cfg.Offset), 1)
		if err != nil {
			l.Error().Err(err).Msgf("read %s failed",cfg.Address)
			return
		} else {
			l.Info().Hex("raw", res).
				Uint16("big_endian", binary.BigEndian.Uint16(res)).
				Uint16("little_endian", binary.LittleEndian.Uint16(res)).
				Msgf("read %s success", cfg.Address)
		}
	case "VD":
		res, err := client.ReadHoldingRegisters(uint16(addr/2-cfg.Offset), 2)
		if err != nil {
			l.Error().Err(err).Msgf("read %s failed",cfg.Address)
			return
		} else {
			evt := l.Info().Hex("raw", res)

			int1 := binary.BigEndian.Uint32(res)
			float1 := math.Float32frombits(int1)

			res = append( res[2:], res[0:2]...)
			int2 :=  binary.BigEndian.Uint32(res)
			float2 := math.Float32frombits(int2)

			evt.Float32("case1", float1).
				Float32("case2", float2).
				Msgf("read %s success", cfg.Address)
		}
	}
}
