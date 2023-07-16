package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/shawn246/tcp-over-http/constant"
)

type agent struct {
	conn   net.Conn
	id     string
	url    string
	target string
}

func newAgent(conn net.Conn, url, target string) *agent {
	return &agent{
		conn:   conn,
		id:     uuid.New().String()[0:8],
		url:    url,
		target: target,
	}
}

func (a *agent) work() {
	log.Info().Str("id", a.id).
		Str("remote", a.conn.RemoteAddr().String()).
		Msgf("new connection established")

	var err error
	ctx := context.Background()

	defer func() {
		a.conn.Close()
		log.Info().
			Str("id", a.id).
			Str("client", a.conn.RemoteAddr().String()).
			Msg("connection closed")
	}()

	if err = a.doProxy(ctx, constant.Establish, nil); err != nil {
		log.Error().Err(err).Str("id", a.id).Msg("proxy failed")
		return
	}

	buff := make([]byte, constant.BuffSize)
	n := 0
	for {
		if n, err = a.conn.Read(buff); err == nil {
			err = a.doProxy(ctx, constant.Forward, buff[0:n])
		}

		if errors.Is(err, io.EOF) {
			_ = a.doProxy(ctx, constant.Goodbye, nil)
			return
		}
		if err != nil {
			log.Error().Err(err).Str("id", a.id).Msg("proxy failed")
			return
		}
	}
}

func (a *agent) doProxy(ctx context.Context, action string, data []byte) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	var reader io.Reader
	if len(data) > 0 {
		log.Debug().Str("id", a.id).Msgf("send %d bytes", len(data))

		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctxTimeout, http.MethodPost, a.url, reader)
	if err != nil {
		return fmt.Errorf("new http request: %w", err)
	}

	req.Header.Set("Proxy-Id", a.id)
	req.Header.Set("Proxy-Target", a.target)
	req.Header.Set("Proxy-Action", action)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("do http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("do http request: status code %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if len(raw) > 0 {
		log.Debug().Str("id", a.id).Msgf("received %d bytes", len(raw))

		if _, err = a.conn.Write(raw); err != nil {
			return fmt.Errorf("tcp write data: %w", err)
		}
	}

	if resp.Header.Get("Proxy-Has-Next") == "true" {
		return a.doProxy(ctx, constant.Require, nil)
	}
	return nil
}
