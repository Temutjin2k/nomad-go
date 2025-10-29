package rabbit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/rabbit"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
)

const (
	RideExchange = "ride_topic"

	QueueDriverResponse     = "driver_responses"
	QueueDriverStatusUpdate = "driver_status"
	QueueLocationUpdate     = "location_updates"
)

type RideBroker struct {
	client       *rabbit.RabbitMQ
	RideExchange string

	l logger.Logger
}

func NewRideBroker(client *rabbit.RabbitMQ, log logger.Logger) *RideBroker {
	rideBroker := &RideBroker{
		client:       client,
		RideExchange: RideExchange,

		l: log,
	}

	return rideBroker
}

// публикует событие о новой поездке для поиска водителя.
// отправляет в exchange 'ride_topic' с ключом 'ride.request.{ride_type}'.
func (r *RideBroker) PublishRideRequested(ctx context.Context, msg models.RideRequestedMessage) error {
	ctx = wrap.WithAction(ctx, "rabbitmq_publish_ride_request")

	// Проверяем и восстанавливаем соединение
	if err := r.client.EnsureConnection(ctx); err != nil {
		r.l.Error(ctx, "ensure connection failed", err)
		return wrap.Error(ctx, err)
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return wrap.Error(ctx, fmt.Errorf("failed to marshal message: %w", err))
	}

	// ключ маршрутизации, example, "ride.request.ECONOMY"
	key := fmt.Sprintf("ride.request.%s", msg.RideType)

	if err := retry(5, time.Second, func() error {
		if err := r.client.Channel.PublishWithContext(
			ctx,
			r.RideExchange, // exchange
			key,            // routing key
			true,           // mandatory
			false,          // immediate
			amqp091.Publishing{
				ContentType:   "application/json",
				CorrelationId: msg.CorrelationID, // для трассировки
				Body:          body,
				Timestamp:     time.Now(),
				Priority:      msg.Priority,
			},
		); err != nil {
			return wrap.Error(ctx, fmt.Errorf("failed to publish with context: %w", err))
		}

		return nil
	}); err != nil {
		return wrap.Error(ctx, err)
	}

	return nil
}

// публикует событие об изменении статуса поездки.
// отправляет в exchange 'ride_topic' с ключом 'ride.status.{status}'.
func (r *RideBroker) PublishRideStatus(ctx context.Context, msg models.RideStatusUpdateMessage) error {
	ctx = wrap.WithAction(ctx, "rabbitmq_publish_ride_status")

	// Проверяем и восстанавливаем соединение
	if err := r.client.EnsureConnection(ctx); err != nil {
		r.l.Error(ctx, "ensure connection failed", err)
		return wrap.Error(ctx, err)
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return wrap.Error(ctx, fmt.Errorf("failed to marshal message: %w", err))
	}

	key := fmt.Sprintf("ride.status.%s", msg.Status)

	if err := retry(5, time.Second, func() error {
		if err := r.client.Channel.PublishWithContext(
			ctx,
			r.RideExchange, // exchange
			key,            // routing key
			false,          // mandatory
			false,          // immediate
			amqp091.Publishing{
				ContentType:   "application/json",
				CorrelationId: msg.CorrelationID,
				Body:          body,
				Timestamp:     time.Now(),
			},
		); err != nil {
			return fmt.Errorf("failed to publish with context: %w", err)
		}

		return nil
	}); err != nil {
		return wrap.Error(ctx, err)
	}

	return nil
}

// DriverStatusUpdateHandler читает обновления статуса от driver сервиса
type DriverStatusUpdateHandler func(ctx context.Context, req models.DriverStatusUpdateMessage) error

func (r *RideBroker) ConsumeDriverStatusUpdate(ctx context.Context, handler DriverStatusUpdateHandler) error {
	ctx = wrap.WithAction(ctx, "rabbitmq_consume_driver_status_update")

	// Основной цикл потребителя
	for {
		if ctx.Err() != nil {
			r.l.Debug(ctx, "consume driver status update stopped by context")
			return nil
		}

		// Проверяем и восстанавливаем соединение
		if err := r.client.EnsureConnection(ctx); err != nil {
			r.l.Error(ctx, "ensure connection failed", err)
			time.Sleep(2 * time.Second)
			continue
		}

		// Подписываемся на очередь
		msgs, err := r.client.Channel.Consume(QueueDriverStatusUpdate, "", false, false, false, false, nil)
		if err != nil {
			r.l.Error(ctx, "consume failed", err)
			time.Sleep(2 * time.Second)
			continue
		}

		r.l.Info(ctx, "start consuming driver status update", "queue", QueueDriverStatusUpdate)

		// Цикл чтения сообщений
	consumeLoop:
		for {
			select {
			case <-ctx.Done():
				r.l.Info(ctx, "driver status update consumer shutting down")
				return nil

			case msg, ok := <-msgs:
				if !ok {
					r.l.Warn(ctx, "message channel closed, reconnecting...")
					time.Sleep(2 * time.Second)
					continue consumeLoop
				}

				go func(d amqp091.Delivery) {
					var req models.DriverStatusUpdateMessage
					if err := json.Unmarshal(d.Body, &req); err != nil {
						r.l.Error(ctx, "failed to unmarshal driver match response", err)
						d.Nack(false, false) // не подтверждаем сообщение
						return
					}

					// добавляем в контекст переменные для логирования и трассировки
					ctxx := wrap.WithRequestID(ctx, d.CorrelationId)

					if err := handler(ctxx, req); err != nil {
						r.l.Error(wrap.ErrorCtx(ctx, err), "failed to handle driver status update", err)

						// если ошибка восстановимая, повторно помещаем в очередь
						if isRecoverableError(err) {
							d.Nack(false, true) // повторно помещаем в очередь
						} else {
							d.Nack(false, false) // не подтверждаем сообщение
						}
					}
				}(msg)
			}
		}
	}
}

type DriverResponseHandler func(ctx context.Context, req models.DriverMatchResponse) error

func (r *RideBroker) ConsumeDriverResponse(ctx context.Context, targetRideID uuid.UUID, handler DriverResponseHandler) error {
	ctx = wrap.WithAction(ctx, "rabbitmq_consume_driver_response")

	// Основной цикл потребителя
	for {
		if ctx.Err() != nil {
			r.l.Debug(ctx, "consume driver response stopped by context")
			return nil
		}

		// Проверяем и восстанавливаем соединение
		if err := r.client.EnsureConnection(ctx); err != nil {
			r.l.Error(ctx, "ensure connection failed", err)
			time.Sleep(2 * time.Second)
			continue
		}

		// Подписываемся на очередь
		msgs, err := r.client.Channel.Consume(QueueDriverResponse, "", false, false, false, false, nil)
		if err != nil {
			r.l.Error(ctx, "consume failed", err)
			time.Sleep(2 * time.Second)
			continue
		}

		r.l.Info(ctx, "start consuming driver response", "queue", QueueDriverResponse)

		// Цикл чтения сообщений
	consumeLoop:
		for {
			select {
			case <-ctx.Done():
				r.l.Info(ctx, "driver response consumer shutting down")
				return errors.New("failed to get response for targetRideID")

			case msg, ok := <-msgs:
				if !ok {
					r.l.Warn(ctx, "message channel closed, reconnecting...")
					time.Sleep(2 * time.Second)
					continue consumeLoop
				}

				var req models.DriverMatchResponse
				if err := json.Unmarshal(msg.Body, &req); err != nil {
					r.l.Error(ctx, "failed to unmarshal driver match response", err)
					msg.Nack(false, false) // не подтверждаем сообщение
					continue consumeLoop
				}

				if req.RideID != targetRideID {
					// Не наш rideID, пропускаем сообщение
					msg.Nack(false, true)
					continue consumeLoop
				}

				// добавляем в контекст переменные для логирования и трассировки
				ctxx := wrap.WithRequestID(wrap.WithRideID(ctx, req.RideID.String()), msg.CorrelationId)

				if err := handler(ctxx, req); err != nil {
					r.l.Error(wrap.ErrorCtx(ctx, err), "failed to handle driver response", err)

					// если ошибка восстановимая, повторно помещаем в очередь
					if isRecoverableError(err) {
						msg.Nack(false, true) // повторно помещаем в очередь
					} else {
						msg.Nack(false, false) // не подтверждаем сообщение
					}
				}

				return nil

			}
		}
	}
}

type LocationUpdateHandler func(ctx context.Context, req models.RideLocationUpdate) error

func (r *RideBroker) ConsumeDriverLocationUpdate(ctx context.Context, handler LocationUpdateHandler) error {
	ctx = wrap.WithAction(ctx, "rabbitmq_consume_driver_location")

	for {
		if ctx.Err() != nil {
			r.l.Debug(ctx, "consume driver location stopped by context")
			return nil
		}

		// Проверяем и восстанавливаем соединение
		if err := r.client.EnsureConnection(ctx); err != nil {
			r.l.Error(ctx, "ensure connection failed", err)
			time.Sleep(2 * time.Second)
			continue
		}

		msgs, err := r.client.Channel.Consume(QueueLocationUpdate, "", false, false, false, false, nil)
		if err != nil {
			r.l.Error(ctx, "consume failed", err)
			time.Sleep(2 * time.Second)
			continue
		}

		r.l.Info(ctx, "start consuming location update", "queue", QueueLocationUpdate)

	consumeLoop:
		for {
			select {
			case <-ctx.Done():
				r.l.Info(ctx, "driver location consumer shutting down")
				return nil

			case msg, ok := <-msgs:
				if !ok {
					r.l.Warn(ctx, "message channel closed, reconnecting...")
					time.Sleep(2 * time.Second)
					continue consumeLoop
				}

				// handle each message in its own goroutine
				go func(d amqp091.Delivery) {
					var req models.RideLocationUpdate
					if err := json.Unmarshal(d.Body, &req); err != nil {
						r.l.Error(ctx, "failed to unmarshal driver location update", err)
						_ = d.Nack(false, false)
						return
					}

					// enrich context for logging/tracing
					if req.RideID == nil {
						// Обработать/Отклонить
						d.Ack(false)
						return
					}

					ctxx := wrap.WithRequestID(wrap.WithRideID(ctx, req.RideID.String()), d.CorrelationId)

					if err := handler(ctxx, req); err != nil {
						r.l.Error(wrap.ErrorCtx(ctx, err), "failed to handle driver location update", err)
						if isRecoverableError(err) {
							_ = d.Nack(false, true) // requeue
						} else {
							_ = d.Nack(false, false) // discard / dead-letter
						}
						return
					}

					// успешно обработано
					if err := d.Ack(false); err != nil {
						r.l.Error(ctx, "failed to ack message", err)
					}
				}(msg)
			}
		}
	}
}
