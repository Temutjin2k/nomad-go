package ws

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/gorilla/websocket"
)

// Conn представляет собой одно соединение WebSocket, связанное с сущностью (например, драйвером)
type Conn struct {
	conn         *websocket.Conn
	entityID     uuid.UUID
	lastActivity time.Time

	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
	l      logger.Logger
}

func NewConn(parent context.Context, entityID uuid.UUID, conn *websocket.Conn, l logger.Logger) *Conn {
	ctx, cancel := context.WithCancel(parent)

	c := &Conn{
		conn:         conn,
		entityID:     entityID,
		lastActivity: time.Now(),
		ctx:          ctx,
		cancel:       cancel,
		l:            l,
	}
	return c
}

// HealthLoop проверяет последнюю активность соединения
// В случае превышения таймаута соединение закрывается
func (c *Conn) HealthLoop(timeout, interval time.Duration) {
	c.l.Debug(c.ctx, "starting health loop",
		"timeout", timeout.String(),
		"interval", interval.String(),
		"entity_ID", c.entityID,
	)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			c.l.Debug(c.ctx, "health loop stopped", "entity_ID", c.entityID)
			return

		case <-ticker.C:
			if c.isIdle(timeout) {
				c.l.Warn(c.ctx, "connection idle too long, closing",
					"idle_for", time.Since(c.lastActivity).String(),
					"timeout", timeout.String(),
					"entity_ID", c.entityID,
				)
				_ = c.Close()
				return
			}
			c.l.Debug(c.ctx, "connection alive", "idle_for", time.Since(c.lastActivity).String(), "entity_ID", c.entityID)
		}
	}
}

// isIdle потокобезопасно проверяет timeout соединения
func (c *Conn) isIdle(timeout time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Since(c.lastActivity) > timeout
}

func (c *Conn) Send(msg map[string]any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.conn.WriteJSON(msg)
}

func (c *Conn) Listen(handler func(msg any) error) error {
	c.l.Debug(c.ctx, "start listening", "entity_ID", c.entityID)

	for {
		select {
		case <-c.ctx.Done():
			return nil

		default:
			var msg map[string]any
			if err := c.conn.ReadJSON(&msg); err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					c.l.Info(c.ctx, "client closed connection normally", "entity_ID", c.entityID)
					return c.Close()
				}
				c.l.Error(c.ctx, "ws message read failed", err, "entity_ID")
				continue
			}

			c.updateLastActivity()
			c.l.Debug(c.ctx, "received message", "entity_ID", c.entityID, "msg", msg)

			if err := handler(msg); err != nil {
				c.l.Error(c.ctx, "failed to handle message", err, "entity_ID", c.entityID)
				continue
			}

			c.l.Debug(c.ctx, "handler OK", "entity_ID", c.entityID)
		}
	}
}

func (c *Conn) updateLastActivity() {
	c.mu.Lock()
	c.lastActivity = time.Now()
	c.mu.Unlock()
}

func (c *Conn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.l.Debug(c.ctx, "closing connection", "entity_ID", c.entityID)

	if c.cancel != nil {
		c.cancel()
		c.l.Debug(c.ctx, "context cancelled", "entity_ID", c.entityID)
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return fmt.Errorf("failed to close websocket: %w", err)
		}
		c.l.Debug(c.ctx, "websocket closed", "entity_ID", c.entityID)
	}

	return nil
}
