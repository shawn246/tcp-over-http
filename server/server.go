package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/shawn246/tcp-over-http/constant"
)

var log *zerolog.Logger

type config struct {
	Port int
	Path string
}

func RegisterLogger(logger *zerolog.Logger) {
	log = logger
}

func RegisterCmd(root *cobra.Command) {
	cfg := &config{}
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "proxy server mode",
		Run: func(cmd *cobra.Command, args []string) {
			run(cmd.Context(), cfg)
		},
	}
	serverCmd.Flags().IntVarP(&cfg.Port, "port", "", 80, "listen port")
	serverCmd.Flags().StringVarP(&cfg.Path, "path", "p", "/proxy", "proxy http path")
	root.AddCommand(serverCmd)
}

func run(ctx context.Context, cfg *config) {
	engine := gin.New()
	engine.GET("/healthz", healthHandler)
	engine.POST("/"+strings.TrimLeft(cfg.Path, "/"), proxyHandler)

	serv := &http.Server{Addr: fmt.Sprintf(":%d", cfg.Port), Handler: engine}
	ctxServ, cancel := context.WithCancel(context.Background())

	go func() {
		defer cancel()

		log.Info().Msgf("proxy server listen at: %d", cfg.Port)
		if err := serv.ListenAndServe(); err != nil {
			log.Error().Err(err).Msgf("proxy server listen failed")
		}
	}()

	select {
	case <-ctx.Done():
		_ = serv.Shutdown(context.Background())
		log.Info().Msgf("proxy server shutdown")
	case <-ctxServ.Done():
	}
}

func healthHandler(c *gin.Context) {
	c.String(http.StatusOK, "ok")
}

func proxyHandler(c *gin.Context) {
	id := c.GetHeader("Proxy-Id")
	action := c.GetHeader("Proxy-Action")
	target := c.GetHeader("Proxy-Target")

	if id == "" || target == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	switch action {
	case constant.Establish:
		ad, err := newAdapter(id, target)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}

		log.Info().Str("id", id).Str("target", target).Msgf("new connection established")

		defer c.Request.Body.Close()
		if err = ad.doProxy(c); err != nil {
			log.Error().Err(err).Str("id", id).Msgf("proxy failed")
			return
		}
	case constant.Forward, constant.Require:
		ad := getAdapter(id)
		if ad == nil {
			c.Status(http.StatusBadRequest)
			return
		}

		defer c.Request.Body.Close()
		if err := ad.doProxy(c); err != nil {
			log.Error().Err(err).Str("id", id).Msgf("proxy failed")
			return
		}
	case constant.Goodbye:
		deleteAdapter(id)
		c.Status(http.StatusOK)
		log.Info().Str("id", id).Msgf("proxy finish")
	default:
		c.Status(http.StatusBadRequest)
	}
}
