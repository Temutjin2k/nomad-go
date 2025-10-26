package ws

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"sync"
	"time"

	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/gorilla/websocket"
)

var ErrListenTimeout = errors.New("listen timeout")

// Conn представляет собой одно соединение WebSocket, связанное с сущностью (например, драйвером)
type Conn struct {
	conn        *websocket.Conn
	entityID    uuid.UUID
	lastPong    time.Time
	subscribers map[string]chan map[string]any

	once   sync.Once
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
	l      logger.Logger
}

func NewConn(parent context.Context, entityID uuid.UUID, conn *websocket.Conn, l logger.Logger) *Conn {
	ctx, cancel := context.WithCancel(parent)

	c := &Conn{
		conn:        conn,
		entityID:    entityID,
		lastPong:    time.Now(),
		subscribers: make(map[string]chan map[string]any),

		ctx:    ctx,
		cancel: cancel,
		l:      l,
	}

	c.conn.SetPongHandler(func(_ string) error {
		c.mu.Lock()
		c.lastPong = time.Now()
		c.mu.Unlock()

		return nil
	})

	return c
}

// Subscribe добавляет новый канал подписки
func (c *Conn) Subscribe(name string, ch chan map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.subscribers[name] = ch
	c.l.Debug(c.ctx, "subscribed", "entity_ID", c.entityID, "subscription", name)
}

// Unsubscribe удаляет подписку
func (c *Conn) Unsubscribe(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.subscribers, name)
	c.l.Debug(c.ctx, "unsubscribed", "entity_ID", c.entityID, "subscription", name)
}

// HeartbeatLoop проверяет последнюю активность соединения
// В случае превышения таймаута соединение закрывается
func (c *Conn) HeartbeatLoop(timeout, interval time.Duration) error {
	c.l.Debug(c.ctx, "starting heartbeat loop",
		"timeout", timeout.String(),
		"interval", interval.String(),
		"entity_ID", c.entityID,
	)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Пингнем сразу при старте
	if err := c.sendPing(); err != nil {
		c.l.Error(c.ctx, "failed to send initial ping", err, "entity_ID", c.entityID)
		return c.Close()
	}

mainLoop:
	for {
		select {
		case <-c.ctx.Done():
			c.l.Debug(c.ctx, "heartbeat loop stopped", "entity_ID", c.entityID)
			break mainLoop
		case <-ticker.C:
			if c.isIdle(timeout) {
				c.l.Warn(c.ctx, "connection idle too long, closing",
					"idle_for", time.Since(c.lastPong).String(),
					"timeout", timeout.String(),
					"entity_ID", c.entityID,
				)
				break mainLoop
			}

			// Отправляем следующий ping и уходим на новый цикл ожидания
			if err := c.sendPing(); err != nil {
				c.l.Error(c.ctx, "failed to send ping", err, "entity_ID", c.entityID)
				return c.Close()
			}
		}
	}
	return c.Close()
}

func (c *Conn) sendPing() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	deadline := time.Now().Add(5 * time.Second)
	if err := c.conn.WriteControl(websocket.PingMessage, nil, deadline); err != nil {
		// На случай специфики транспорта — fallback
		return c.conn.WriteMessage(websocket.PingMessage, nil)
	}
	return nil
}

// Listen читает сообщения и рассылает подписчикам
func (c *Conn) Listen() error {
	c.l.Debug(c.ctx, "start listening", "entity_ID", c.entityID)

mainLoop:
	for {
		select {
		case <-c.ctx.Done():
			c.l.Debug(c.ctx, "listen stopped (context cancelled)", "entity_ID", c.entityID)
			break mainLoop

		default:
			var msg map[string]any
			if err := c.conn.ReadJSON(&msg); err != nil {
				if websocket.IsCloseError(err,
					websocket.CloseNormalClosure,
					websocket.CloseGoingAway,
					websocket.CloseAbnormalClosure) ||
					errors.Is(err, net.ErrClosed) ||
					errors.Is(err, io.EOF) {
					c.l.Info(c.ctx, "websocket closed", "entity_ID", c.entityID)
					break mainLoop
				}
				c.l.Error(c.ctx, "failed to read ws message", err, "entity_ID", c.entityID)
				continue
			}

			c.mu.Lock()
			c.lastPong = time.Now()
			subs := make(map[string]chan map[string]any, len(c.subscribers))
			maps.Copy(subs, c.subscribers)
			c.mu.Unlock()

			c.l.Debug(c.ctx, "received message", "entity_ID", c.entityID, "msg", msg)

			for name, ch := range subs {
				go func(name string, ch chan map[string]any, msg map[string]any) {
					select {
					case ch <- msg:
						c.l.Debug(c.ctx, "message broadcasted", "entity_ID", c.entityID, "subscription", name)
					case <-time.After(100 * time.Millisecond):
						c.l.Warn(c.ctx, "broadcast timeout, dropping message", "entity_ID", c.entityID, "subscription", name)
					case <-c.ctx.Done():
						c.l.Debug(c.ctx, "listen stopped (context cancelled)", "entity_ID", c.entityID)
						return
					}
				}(name, ch, msg)
			}
		}
	}
	return c.Close()
}

// isIdle потокобезопасно проверяет timeout соединения
func (c *Conn) isIdle(timeout time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Since(c.lastPong) > timeout
}

func (c *Conn) Send(msg any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.conn.WriteJSON(msg)
}

func (c *Conn) Close() error {
	var err error
	c.once.Do(func() {
		c.mu.Lock()
		defer c.mu.Unlock()

		c.l.Debug(c.ctx, "closing connection", "entity_ID", c.entityID)

		if c.cancel != nil {
			c.cancel()
			c.l.Debug(c.ctx, "context cancelled", "entity_ID", c.entityID)
		}

		if c.conn != nil {
			if e := c.conn.Close(); e != nil {
				err = fmt.Errorf("failed to close websocket: %w", e)
			} else {
				c.l.Debug(c.ctx, "websocket closed", "entity_ID", c.entityID)
			}
			c.conn = nil
		}

		for name := range c.subscribers {
			delete(c.subscribers, name)
		}
	})
	return err
}
