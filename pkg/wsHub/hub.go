package ws

import (
	"context"
	"errors"
	"maps"
	"sync"

	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

var (
	ErrEmptyConn       = errors.New("connection is empty")
	ErrConnIsNotFound  = errors.New("connection not found")
	maxPendingMessages = 64
)

type pendingMsg struct {
	Data any
}

// ConnectionHub хранит и управляет всеми активными WebSocket соединениями
type ConnectionHub struct {
	clients map[uuid.UUID]*Conn
	pending map[uuid.UUID][]pendingMsg // буфер непросланных сообщений

	l  logger.Logger
	mu sync.Mutex
	wg sync.WaitGroup
}

func NewConnHub(l logger.Logger) *ConnectionHub {
	return &ConnectionHub{
		clients: make(map[uuid.UUID]*Conn),
		pending: make(map[uuid.UUID][]pendingMsg),
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

	go h.OnReconnect(newConn.entityID)

	return nil
}

// OnReconnect вызывается при новом подключении клиента.
// Отправляет все отложенные (pending) сообщения, если они есть.
func (h *ConnectionHub) OnReconnect(id uuid.UUID) {
	h.mu.Lock()
	pending, ok := h.pending[id]
	conn, connOK := h.clients[id]
	h.mu.Unlock()

	if !ok || !connOK || len(pending) == 0 {
		return // нечего восстанавливать
	}

	ctx := wrap.WithAction(context.Background(), "ws_on_reconnect")
	h.l.Info(ctx, "resending pending messages", "entity_ID", id, "count", len(pending))

	// последовательно отсылаем буфер
	for _, msg := range pending {
		if err := conn.Send(msg.Data); err != nil {
			h.l.Warn(ctx, "failed to resend pending message", "entity_ID", id, "err", err.Error())
			break // прерываем, если соединение снова умерло
		}
	}

	// если дошли до сюда — очистим буфер
	h.mu.Lock()
	delete(h.pending, id)
	h.mu.Unlock()

	h.l.Info(ctx, "pending messages delivered and cleared", "entity_ID", id)
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

func (h *ConnectionHub) cachePending(id uuid.UUID, msg any) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.pending == nil {
		h.pending = make(map[uuid.UUID][]pendingMsg)
	}

	pending := h.pending[id]
	if len(pending) >= maxPendingMessages {
		// удаляем самое старое
		pending = pending[1:]
	}

	// добавляем новое сообщение
	pending = append(pending, pendingMsg{Data: msg})
	h.pending[id] = pending
}

// SendTo отправляет сообщение определённому клиенту по ID
// возвращает ошибку ErrConnIsNotFound, если соединение не найдена
func (h *ConnectionHub) SendTo(id uuid.UUID, msg any) error {
	h.mu.Lock()
	conn, ok := h.clients[id]
	h.mu.Unlock()

	if !ok {
		// нет соединения — кешируем сообщение
		h.cachePending(id, msg)
		return ErrConnIsNotFound
	}

	if err := conn.Send(msg); err != nil {
		// соединение могло отвалиться в момент отправки
		h.cachePending(id, msg)
		return err
	}

	return nil
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
	maps.Copy(copyMap, h.clients)
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
