package ws

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

var ErrEmptyConn = errors.New("connection is empty")

type ConnectionHub struct {
	clients map[uuid.UUID]*Conn
	l       logger.Logger
	mu      sync.Mutex
	wg      sync.WaitGroup
}

func NewConnHub(l logger.Logger) *ConnectionHub {
	return &ConnectionHub{
		clients: map[uuid.UUID]*Conn{},
		l:       l,
	}
}

func (h *ConnectionHub) Add(new *Conn) error {
	if new == nil {
		return ErrEmptyConn
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	if val, ok := h.clients[new.entityID]; ok {
		if err := val.Close(); err != nil {
			h.l.Warn(context.Background(), "failed to close existing conn", "entity_ID", val.entityID, "err", err.Error())
		}
	}

	h.clients[new.entityID] = new
	h.wg.Add(1)
	return nil
}

func (h *ConnectionHub) Delete(entityID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if val, ok := h.clients[entityID]; ok {
		if err := val.Close(); err != nil {
			h.l.Warn(context.Background(), "failed to close conn", "entity_ID", val.entityID, "err", err.Error())
		}
		delete(h.clients, entityID)
		h.wg.Done()
	}
}

func (h *ConnectionHub) SendTo(id uuid.UUID, msg map[string]any) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conn, ok := h.clients[id]; ok {
		return conn.Send(msg)
	}
	return fmt.Errorf("no connection for %s", id)
}

func (h *ConnectionHub) Close() {
	h.mu.Lock()
	for _, conn := range h.clients {
		h.Delete(conn.entityID)
	}
	h.mu.Unlock()
	h.wg.Wait()
}

func (h *ConnectionHub) HealthLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			h.l.Info(ctx, "health loop stopped")
			return
		case <-ticker.C:
			h.mu.Lock()
			for id, conn := range h.clients {
				if err := conn.Health(); err != nil {
					h.l.Warn(ctx, "dead connection", "id", id, "err", err.Error())
					h.Delete(id)
				}
			}
			h.mu.Unlock()
		}
	}
}
