package server

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/shawn246/tcp-over-http/constant"
)

var (
	adapters = make(map[string]*adapter)
	mutex    sync.RWMutex
)

type adapter struct {
	id     string
	conn   net.Conn
}

func getAdapter(id string) *adapter {
	mutex.RLock()
	defer mutex.RUnlock()
	return adapters[id]
}

func deleteAdapter(id string) {
	mutex.Lock()
	defer mutex.Unlock()

	if ad := adapters[id]; ad != nil {
		ad.shutdown()
	}
}

func newAdapter(id, target string) (*adapter, error) {
	conn, err := net.DialTimeout("tcp", target, time.Second*5)
	if err != nil {
		return nil, fmt.Errorf("dail %s: %w", target, err)
	}

	mutex.Lock()
	defer mutex.Unlock()

	ad := &adapter{
		id:   id,
		conn: conn,
	}

	adapters[id] = ad
	return ad, nil
}

func (a *adapter) write(c *gin.Context) error {
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return fmt.Errorf("read request body: %w", err)
	} else if len(raw) == 0 {
		return nil
	}

	if _, err = a.conn.Write(raw); err != nil {
		return fmt.Errorf("tcp write data: %w", err)
	}

	log.Debug().Str("id", a.id).Msgf("received %d bytes", len(raw))
	return nil
}

func (a *adapter) read(c *gin.Context) error {
	var data []byte
	var err error
	var n int

	buff := make([]byte, constant.BuffSize)
	failed := 0

	for i := 0; i < 100; i++ {
		if failed >= 3 {
			break
		}

		dur := time.Millisecond * 100
		if i == 0 {
			dur = time.Minute * 5
		}

		_ = a.conn.SetReadDeadline(time.Now().Add(dur))
		if n, err = a.conn.Read(buff); err == nil {
			data = append(data, buff[0:n]...)
		} else if errors.Is(err, os.ErrDeadlineExceeded) || errors.Is(err, io.EOF) {
			err = nil
			failed += 1
		} else {
			break
		}
	}

	if len(data) > 0 {
		c.Data(http.StatusOK, "application/octet-stream", data)
		log.Debug().Str("id", a.id).Msgf("send %d bytes", len(data))
	}
	return err
}

func (a *adapter) shutdown() {
	_ = a.conn.Close()
}
