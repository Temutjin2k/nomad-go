package ws

import (
	"context"
	"errors"
	"sync"

	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

var (
	ErrEmptyConn      = errors.New("connection is empty")
	ErrConnIsNotFound = errors.New("connection not found")
)

// ConnectionHub хранит и управляет всеми активными WebSocket соединениями
type ConnectionHub struct {
	clients map[uuid.UUID]*Conn
	l       logger.Logger
	mu      sync.Mutex
	wg      sync.WaitGroup
}

func NewConnHub(l logger.Logger) *ConnectionHub {
	return &ConnectionHub{
		clients: make(map[uuid.UUID]*Conn),
		l:       l,
	}
}

// Add добавляет новое соединение в хаб.
// Если соединение с этим entityID уже существует — оно закрывается.
func (h *ConnectionHub) Add(newConn *Conn) error {
	if newConn == nil {
		return ErrEmptyConn
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	ctx := wrap.WithAction(context.Background(), "add_ws_connection")

	if existing, ok := h.clients[newConn.entityID]; ok {
		h.l.Warn(ctx,
			"replacing existing connection",
			"entity_ID", existing.entityID,
		)
		if err := existing.Close(); err != nil {
			h.l.Warn(ctx,
				"failed to close existing conn",
				"entity_ID", existing.entityID,
				"err", err.Error(),
			)
		}
	}

	h.clients[newConn.entityID] = newConn
	h.wg.Add(1)

	return nil
}

// Delete удаляет и закрывает соединение по ID
func (h *ConnectionHub) Delete(entityID uuid.UUID) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	ctx := wrap.WithAction(context.Background(), "ws_connection_delete")

	conn, ok := h.clients[entityID]
	if !ok {
		h.l.Warn(ctx,
			"delete called for unknown entity",
			"entity_ID", entityID,
		)
		return ErrConnIsNotFound
	}

	if err := conn.Close(); err != nil {
		h.l.Warn(ctx,
			"failed to close conn",
			"entity_ID", conn.entityID,
			"err", err.Error(),
		)
	}

	delete(h.clients, entityID)
	h.wg.Done()

	return nil
}

// SendTo отправляет сообщение определённому клиенту по ID
// возвращает ошибку ErrConnIsNotFound, если соединение не найдена
func (h *ConnectionHub) SendTo(id uuid.UUID, msg map[string]any) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conn, ok := h.clients[id]; ok {
		return conn.Send(msg)
	}
	return ErrConnIsNotFound
}

// Close закрывает каждое websocket соединение
func (h *ConnectionHub) Close() {
	ctx := wrap.WithAction(context.Background(), "hub_close")

	// копируем клиентов под локом
	h.mu.Lock()
	clients := make([]*Conn, 0, len(h.clients))
	for _, conn := range h.clients {
		clients = append(clients, conn)
	}
	h.mu.Unlock()
	// закрываем вне локов
	for _, conn := range clients {
		_ = h.Delete(conn.entityID)
	}

	h.wg.Wait()

	h.l.Info(ctx, "all websocket connections closed gracefully")
}

// Clients возвращает копию списка клиентов
func (h *ConnectionHub) Clients() map[uuid.UUID]*Conn {
	h.mu.Lock()
	defer h.mu.Unlock()

	copyMap := make(map[uuid.UUID]*Conn, len(h.clients))
	for id, conn := range h.clients {
		copyMap[id] = conn
	}
	return copyMap
}

// GetConn возвращает нужное соединение по UUID
func (h *ConnectionHub) GetConn(id uuid.UUID) (*Conn, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	conn, ok := h.clients[id]
	if !ok {
		return nil, ErrConnIsNotFound
	}
	return conn, nil
}
