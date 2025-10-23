package rabbit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	Conn      *amqp.Connection
	Channel   *amqp.Channel
	closeChan chan *amqp.Error
	isClosed  bool
	mu        sync.Mutex
	dsn       string

	log logger.Logger
}

// New creates rabbitMQ client
func New(ctx context.Context, dsn string, log logger.Logger) (*RabbitMQ, error) {
	conn, err := amqp.DialConfig(dsn, amqp.Config{
		Heartbeat: 10 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	// Create a channel
	channel, err := conn.Channel()
	if err != nil {
		conn.Close() // Close connection if channel creation fails
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	// Create close notification channel
	closeChan := make(chan *amqp.Error, 1)
	conn.NotifyClose(closeChan)

	// Verify the connection is alive
	select {
	case closeErr := <-closeChan:
		if closeErr != nil {
			return nil, fmt.Errorf("rabbitmq connection is closed: %w", closeErr)
		}
		return nil, fmt.Errorf("rabbitmq connection is closed")
	default:
		// Connection is good
	}

	log.Info(ctx, types.ActionRabbitMQConnected, "connected to rabbitMQ")

	r := &RabbitMQ{
		Conn:      conn,
		Channel:   channel,
		closeChan: closeChan,
		isClosed:  false,
		dsn:       dsn,
		log:       log,
	}

	// Start monitoring connection in background
	go r.monitorConnection()

	return r, nil
}

// monitorConnection monitors the connection status
func (r *RabbitMQ) monitorConnection() {
	closeErr := <-r.closeChan
	r.isClosed = true

	ctx := wrap.WithAction(context.Background(), types.ActionRabbitConnectionClosed)

	if closeErr != nil {
		r.log.Error(ctx, "RabbitMQ connection closed with error", closeErr)
	} else {
		r.log.Debug(ctx, "RabbitMQ connection closed gracefully")
	}
}

// IsConnectionClosed checks if the connection is closed
func (r *RabbitMQ) IsConnectionClosed() bool {
	if r.Conn == nil {
		return true
	}
	return r.isClosed || r.Conn.IsClosed()
}

// Close closes rabbit connection
func (r *RabbitMQ) Close(ctx context.Context) error {
	return r.closeWithContext(ctx)
}

// closeWithContext - closes RabbitMQ channel and connection using context
func (r *RabbitMQ) closeWithContext(ctx context.Context) error {
	ctx = wrap.WithAction(ctx, types.ActionRabbitConnectionClosing)

	r.log.Debug(ctx, "closing channel")

	// quick check under lock
	r.mu.Lock()
	if r.isClosed {
		r.mu.Unlock()
		return nil
	}
	// mark closed early to avoid races with concurrent Close calls
	r.isClosed = true
	ch := r.Channel
	conn := r.Conn
	// Clear references so other goroutines know it's closed
	r.Channel = nil
	r.Conn = nil
	r.mu.Unlock()

	// Close channel first (if any)
	if ch != nil {
		if err := closeWithCtxFunc(ctx, ch.Close); err != nil {
			// If context cancelled, log it; if Close itself failed, log and continue to try closing conn
			if ctx.Err() != nil {
				r.log.Debug(ctx, "context cancelled while closing channel")
			} else {
				r.log.Error(ctx, "error closing channel", err)
			}
		}
	}

	r.log.Debug(ctx, "closing RabbitMQ connection")

	// Close connection
	if conn != nil {
		if err := closeWithCtxFunc(ctx, conn.Close); err != nil {
			if ctx.Err() != nil {
				r.log.Debug(ctx, "context cancelled while closing connection")
				// prefer to return context error
				return ctx.Err()
			}
			return fmt.Errorf("failed to close connection: %w", err)
		}
	}

	ctx = wrap.WithAction(ctx, types.ActionRabbitConnectionClosed)
	r.log.Info(ctx, "rabbitMQ closed")

	return nil
}

// helper to close a resource with context cancellation safely
func closeWithCtxFunc(ctx context.Context, fn func() error) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- fn()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		// Return context error; goroutine can still write into the buffered channel and exit.
		return ctx.Err()
	}
}

func (r *RabbitMQ) Reconnect(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.dsn == "" {
		return fmt.Errorf("dsn is empty: can't reconnect")
	}

	if !r.isClosed && r.Conn != nil && !r.Conn.IsClosed() {
		return nil
	}

	var conn *amqp.Connection
	var err error

	for i := 0; i < 5; i++ {
		conn, err = amqp.DialConfig(r.dsn, amqp.Config{
			Heartbeat: 10 * time.Second,
		})
		if err == nil {
			break
		}

		wait := time.Duration(i+1) * 2 * time.Second
		r.log.Debug(ctx, fmt.Sprintf("reconnect attempt %d failed, retrying in %v", i+1, wait))

		select {
		case <-ctx.Done():
			r.log.Debug(ctx, "graceful shutdown — stopping reconnect attempts")
			return ctx.Err()
		case <-time.After(wait):
		}
	}

	if err != nil {
		return fmt.Errorf("failed to reconnect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open a channel after reconnect: %w", err)
	}

	closeChan := make(chan *amqp.Error, 1)
	conn.NotifyClose(closeChan)

	r.Conn = conn
	r.Channel = ch
	r.closeChan = closeChan
	r.isClosed = false

	// Возобновляем мониторинг
	go r.monitorConnection()

	ctx = wrap.WithAction(context.Background(), types.ActionRabbitReconnected)
	r.log.Info(ctx, "RabbitMQ reconnected successfully")

	return nil
}
