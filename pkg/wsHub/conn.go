package ws

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/gorilla/websocket"
)

type Conn struct {
	conn     *websocket.Conn
	entityID uuid.UUID
	doneCtx  context.Context
	cancel   context.CancelFunc
	mu       sync.Mutex
}

func NewConn(ctx context.Context, entityID uuid.UUID, conn *websocket.Conn) *Conn {
	ctx, cancel := context.WithCancel(ctx)

	return &Conn{
		conn:     conn,
		entityID: entityID,
		doneCtx:  ctx,
		cancel:   cancel,
	}
}

func (c *Conn) Health() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return errors.New("connection is nil")
	}

	select {
	case <-c.doneCtx.Done():
		return errors.New("connection context cancelled")
	default:
	}

	if err := c.conn.WriteControl(
		websocket.PingMessage,
		[]byte("ping"),
		time.Now().Add(3*time.Second),
	); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}

func (c *Conn) Send(msg map[string]any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.Health(); err != nil {
		return fmt.Errorf("send failed: connection not healthy: %w", err)
	}
	return c.conn.WriteJSON(msg)
}

func (c *Conn) Listen(handler func(msg any) error) error {
	for {
		select {
		case <-c.doneCtx.Done():
			return errors.New("listen stopped: context done")
		default:
			var msg map[string]any
			if err := c.conn.ReadJSON(&msg); err != nil {
				return fmt.Errorf("read failed: %w", err)
			}
			if err := handler(msg); err != nil {
				return fmt.Errorf("handler failed: %w", err)
			}
		}
	}
}

func (c *Conn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
