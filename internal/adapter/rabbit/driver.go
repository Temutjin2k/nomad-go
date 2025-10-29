package rabbit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/internal/domain/types"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/rabbit"
)

const (
	ExchangeDriverTopic    = "driver_topic"
	ExchangeLocationFanout = "location_fanout"

	QueueRideStatus   = "ride_status"
	QueueRideRequests = "ride_requests"
)

type DriverBroker struct {
	client        *rabbit.RabbitMQ
	exchangeTypes map[string]string
	l             logger.Logger
}

func NewDriverClient(client *rabbit.RabbitMQ, l logger.Logger) *DriverBroker {
	p := &DriverBroker{
		client: client,
		exchangeTypes: map[string]string{
			ExchangeDriverTopic:    "topic",
			ExchangeLocationFanout: "fanout",
		},
		l: l,
	}
	return p
}

func (r *DriverBroker) publish(ctx context.Context, exchange, routingKey string, msg any) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	pub := amqp.Publishing{
		ContentType:   "application/json",
		Body:          body,
		Timestamp:     time.Now(),
		CorrelationId: wrap.GetRequestID(ctx),
	}

	if err := retry(5, time.Second*2,
		func() error {
			return r.client.Channel.PublishWithContext(
				ctx,
				exchange,
				routingKey,
				false,
				false,
				pub,
			)
		}); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}

func (r *DriverBroker) PublishDriverStatus(ctx context.Context, msg models.DriverStatusUpdateMessage) error {
	ctx = wrap.WithAction(ctx, "publish_driver_status")
	key := fmt.Sprintf("driver.status.%s", msg.DriverID)

	if err := r.publish(ctx, ExchangeDriverTopic, key, msg); err != nil {
		return wrap.Error(ctx, err)
	}
	return nil
}

func (r *DriverBroker) PublishDriverResponse(ctx context.Context, msg models.DriverMatchResponse) error {
	ctx = wrap.WithAction(ctx, "publish_driver_response")
	key := fmt.Sprintf("driver.response.%s", msg.RideID)

	if err := r.publish(ctx, ExchangeDriverTopic, key, msg); err != nil {
		return wrap.Error(ctx, err)
	}
	return nil
}

func (r *DriverBroker) PublishLocationUpdate(ctx context.Context, msg models.RideLocationUpdate) error {
	ctx = wrap.WithAction(ctx, "publish_location_update")
	key := "" // N/A

	if err := r.publish(ctx, ExchangeLocationFanout, key, msg); err != nil {
		return wrap.Error(ctx, err)
	}
	return nil
}

// -- Consumers

type ConsumeRideHandlerFunc func(ctx context.Context, req models.RideRequestedMessage) error

// ConsumeRideRequest слушает ride.request.* события и передаёт их в обработчик fn.
func (r *DriverBroker) ConsumeRideRequest(ctx context.Context, fn ConsumeRideHandlerFunc) error {
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

		// Подписываемся на очередь
		msgs, err := r.client.Channel.Consume(QueueRideRequests, "", false, false, false, false, nil)
		if err != nil {
			r.l.Error(ctx, "consume failed", err, "op", op)
			time.Sleep(2 * time.Second)
			continue
		}

		r.l.Info(ctx, "start consuming ride requests", "queue", QueueRideRequests)

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
					break consumeLoop
				}

				go r.handleRideRequested(ctx, fn, msg)
			}
		}
	}
}

type MatchConfHandlerFunc func(ctx context.Context, req models.RideStatusUpdateMessage, stopCh chan struct{}) error

func (r *DriverBroker) ConsumeStatusUpdate(ctx context.Context, fn MatchConfHandlerFunc) error {
	const op = "RideConsumer.ConsumeStatusUpdate"

	for {
		if ctx.Err() != nil {
			r.l.Debug(ctx, "consume status update stopped by context")
			return nil
		}

		// Проверяем и восстанавливаем соединение
		if err := r.client.EnsureConnection(ctx); err != nil {
			r.l.Error(ctx, "ensure connection failed", err, "op", op)
			time.Sleep(2 * time.Second)
			continue
		}

		// Подписываемся на очередь
		msgs, err := r.client.Channel.Consume(QueueRideStatus, "", false, false, false, false, nil)
		if err != nil {
			r.l.Error(ctx, "consume failed", err, "op", op)
			time.Sleep(2 * time.Second)
			continue
		}

		r.l.Info(ctx, "start consuming ride status", "queue", QueueRideStatus)

		stopCh := make(chan struct{}, 1)
	consumeLoop:
		for {
			select {
			case <-ctx.Done():
				r.l.Info(ctx, "match confirmation consumer shutting down", "op", op)
				return nil
			case <-stopCh:
				r.l.Info(ctx, "match confirmation consumer has been stopped", "op", op)
				return nil

			case msg, ok := <-msgs:
				if !ok {
					r.l.Warn(ctx, "message channel closed, reconnecting...", "op", op)
					time.Sleep(2 * time.Second)
					break consumeLoop
				}

				// Обрабатываем сообщение
				go func(msg amqp.Delivery) {
					var req models.RideStatusUpdateMessage
					if err := json.Unmarshal(msg.Body, &req); err != nil {
						r.l.Error(ctx, "decode failed", err, "op", op)
						_ = msg.Nack(false, false)
						return
					}

					ctxx := wrap.WithRequestID(wrap.WithRideID(ctx, req.RideID.String()), msg.CorrelationId)

					// Вызов обработчика
					if err := fn(ctxx, req, stopCh); err != nil {
						r.l.Error(ctx, "failed to handle status update", err, "op", op)
						_ = msg.Nack(false, false)
						return
					}

					if err := msg.Ack(false); err != nil {
						r.l.Warn(ctx, "ack failed", err, "op", op)
					}
				}(msg)
			}
		}
	}
}

func (r *DriverBroker) handleRideRequested(ctx context.Context, fn ConsumeRideHandlerFunc, msg amqp.Delivery) {
	ctx = wrap.WithAction(ctx, "rabbitmq_handle_ride_requested")

	var req models.RideRequestedMessage
	if err := json.Unmarshal(msg.Body, &req); err != nil {
		r.l.Error(ctx, "decode failed", err)
		_ = msg.Nack(false, false)
		return
	}

	ctxx := wrap.WithRequestID(wrap.WithRideID(ctx, req.RideType), msg.CorrelationId)

	// Вызываем бизнес-обработчик
	if err := fn(ctxx, req); err != nil {
		r.l.Error(ctx, "failed to handle ride request", err)

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
		r.l.Warn(ctx, "ack failed", err)
	}
}
