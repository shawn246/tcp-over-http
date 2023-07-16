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
	id   string
	conn net.Conn
	buff []byte
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
		buff: make([]byte, constant.BuffSize),
	}
	adapters[id] = ad

	return ad, nil
}

func (a *adapter) doProxy(c *gin.Context) error {
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return fmt.Errorf("read request body: %w", err)
	}

	if len(raw) > 0 {
		log.Debug().Str("id", a.id).Msgf("received %d bytes", len(raw))

		if _, err = a.conn.Write(raw); err != nil {
			return fmt.Errorf("tcp write data: %w", err)
		}
	}

	data, err := a.read()
	if err == nil {
		c.Writer.Header().Set("Proxy-Has-Next", "true")
	} else if !errors.Is(err, os.ErrDeadlineExceeded) {
		return fmt.Errorf("tcp read data: %w", err)
	}

	c.Data(http.StatusOK, "application/octet-stream", data)
	log.Debug().Str("id", a.id).Msgf("send %d bytes", len(data))
	return nil
}

func (a *adapter) read() ([]byte, error) {
	var data []byte
	buff := make([]byte, constant.BuffSize)
	for i := 0; i < 3; i++ {
		_ = a.conn.SetReadDeadline(time.Now().Add(time.Millisecond * 100))
		if n, err := a.conn.Read(buff); err == nil {
			data = append(data, buff[0:n]...)
		} else {
			return data, err
		}
	}

	return data, nil
}

func (a *adapter) shutdown() {
	_ = a.conn.Close()
}
