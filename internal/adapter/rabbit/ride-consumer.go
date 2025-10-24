package rabbit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/rabbit"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RideConsumer struct {
	client *rabbit.RabbitMQ
	l      logger.Logger
}

func NewRideConsumer(client *rabbit.RabbitMQ, l logger.Logger) *RideConsumer {
	return &RideConsumer{client: client, l: l}
}

type HandlerFunc func(ctx context.Context, req models.RideRequestedMessage) error

// declareAndBindQueue объявляет и привязывает очередь к exchange.
func (r *RideConsumer) declareAndBindQueue(ctx context.Context, queueName, bindingKey, exchangeName string) (amqp.Queue, error) {
	const op = "RideConsumer.declareAndBindQueue"

	q, err := r.client.Channel.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return q, wrap.Error(ctx, fmt.Errorf("%s: declare queue failed: %w", op, err))
	}

	if err := r.client.Channel.QueueBind(q.Name, bindingKey, exchangeName, false, nil); err != nil {
		return q, wrap.Error(ctx, fmt.Errorf("%s: bind queue failed: %w", op, err))
	}

	return q, nil
}

func (r *RideConsumer) handleMessage(ctx context.Context, fn HandlerFunc, msg amqp.Delivery) {
	const op = "RideConsumer.handleMessage"

	var req models.RideRequestedMessage
	if err := json.Unmarshal(msg.Body, &req); err != nil {
		r.l.Error(ctx, "decode failed", err, "op", op)
		_ = msg.Nack(false, false)
		return
	}

	// Вызываем бизнес-обработчик
	if err := fn(ctx, req); err != nil {
		r.l.Error(ctx, "handler failed", err, "op", op)

		// Если водителей нет — это не ошибка, просто игнор
		if errors.Is(err, types.ErrDriversNotFound) || errors.Is(err, types.ErrDriverSearchTimeout) {
			r.l.Warn(ctx, "dropping message", "reason", err.Error())
			_ = msg.Reject(false)
			return
		}

		_ = msg.Nack(false, false)
		return
	}

	// Успешно обработано
	if err := msg.Ack(false); err != nil {
		r.l.Warn(ctx, "ack failed", err, "op", op)
	}
}

// ConsumeRideRequest слушает ride.request.* события и передаёт их в обработчик fn.
func (r *RideConsumer) ConsumeRideRequest(ctx context.Context, fn HandlerFunc) error {
	const op = "RideConsumer.ConsumeRideRequest"

	// Основной цикл потребителя
	for {
		if ctx.Err() != nil {
			r.l.Debug(ctx, "consume ride request stopped by context")
			return nil
		}

		// Проверяем и восстанавливаем соединение
		if err := r.client.EnsureConnection(ctx); err != nil {
			r.l.Error(ctx, "ensure connection failed", err, "op", op)
			time.Sleep(2 * time.Second)
			continue
		}

		// Гарантируем наличие exchange
		if err := r.client.Channel.ExchangeDeclare("ride_topic", "topic", true, false, false, false, nil); err != nil {
			r.l.Error(ctx, "declare exchange failed", err, "op", op)
			time.Sleep(3 * time.Second)
			continue
		}

		// Объявляем и биндим очередь
		q, err := r.declareAndBindQueue(ctx, "ride_requests", "ride.request.*", "ride_topic")
		if err != nil {
			r.l.Error(ctx, "declare queue failed", err, "op", op)
			time.Sleep(2 * time.Second)
			continue
		}

		// Подписываемся на очередь
		msgs, err := r.client.Channel.Consume(q.Name, "", false, false, false, false, nil)
		if err != nil {
			r.l.Error(ctx, "consume failed", err, "op", op)
			time.Sleep(2 * time.Second)
			continue
		}

		r.l.Info(ctx, "start consuming ride requests", "queue", q.Name)

		// Цикл чтения сообщений
	consumeLoop:
		for {
			select {
			case <-ctx.Done():
				r.l.Info(ctx, "ride request consumer shutting down", "op", op)
				return nil

			case msg, ok := <-msgs:
				if !ok {
					r.l.Warn(ctx, "message channel closed, reconnecting...", "op", op)
					time.Sleep(2 * time.Second)
					continue consumeLoop
				}

				go r.handleMessage(ctx, fn, msg)
			}
		}
	}
}

func (r *RideConsumer) ConsumeMatchConfirmation(ctx context.Context) error {
	const op = "RideConsumer.ConsumeMatchConfirmation"
	return nil
}
