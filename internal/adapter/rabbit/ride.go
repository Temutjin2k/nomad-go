package rabbit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/rabbit"
)

const (
	RideExchange = "ride_topic"

	QueueDriverResponse     = ""
	QueueDriverStatusUpdate = ""
	QueueLocationUpdate     = ""
	QueueRideStatusUpdate   = ""
)

type RideMsgBroker struct {
	client       *rabbit.RabbitMQ
	RideExchange string

	QueueDriverResponse     string
	QueueDriverStatusUpdate string
	QueueLocationUpdate     string
	QueueRideStatusUpdate   string

	l logger.Logger
}

func NewRideMsgBroker(client *rabbit.RabbitMQ, log logger.Logger) *RideMsgBroker {
	rideBroker := &RideMsgBroker{
		client:       client,
		RideExchange: RideExchange,

		QueueDriverResponse:     QueueDriverResponse,
		QueueDriverStatusUpdate: QueueDriverStatusUpdate,
		QueueLocationUpdate:     QueueLocationUpdate,
		QueueRideStatusUpdate:   QueueRideStatusUpdate,

		l: log,
	}

	return rideBroker
}

// публикует событие о новой поездке для поиска водителя.
// отправляет в exchange 'ride_topic' с ключом 'ride.request.{ride_type}'.
func (r *RideMsgBroker) PublishRideRequested(ctx context.Context, msg models.RideRequestedMessage) error {
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
}

// публикует событие об изменении статуса поездки.
// отправляет в exchange 'ride_topic' с ключом 'ride.status.{status}'.
func (r *RideMsgBroker) PublishRideStatus(ctx context.Context, msg models.RideStatusUpdateMessage) error {
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
		return wrap.Error(ctx, fmt.Errorf("failed to publish with context: %w", err))
	}

	return nil
}

// TODO: подумать нужно ли это вообще
type RideStatusUpdateHandler func(ctx context.Context, req models.RideStatusUpdateMessage) error

func (r *RideMsgBroker) ConsumeRideStatusUpdate(ctx context.Context, handler RideStatusUpdateHandler) error {
	return nil
}

// TODO: подумать нужно ли это вообще
type DriverStatusUpdateHandler func(ctx context.Context, req models.DriverStatusUpdateMessage) error

func (r *RideMsgBroker) ConsumeDriverStatusUpdate(ctx context.Context, handler DriverStatusUpdateHandler) error {
	return nil
}

type DriverResponseHandler func(ctx context.Context, req models.DriverMatchResponse) error

func (r *RideMsgBroker) ConsumeDriverResponse(ctx context.Context, handler DriverResponseHandler) error {
	ctx = wrap.WithAction(ctx, "rabbitmq_consume_driver_response")

	// Основной цикл потребителя
	for {
		if ctx.Err() != nil {
			r.l.Debug(ctx, "consume ride request stopped by context")
			return nil
		}

		// Проверяем и восстанавливаем соединение
		if err := r.client.EnsureConnection(ctx); err != nil {
			r.l.Error(ctx, "ensure connection failed", err)
			time.Sleep(2 * time.Second)
			continue
		}

		// Подписываемся на очередь
		const queue = "driver_topic"
		msgs, err := r.client.Channel.Consume(r.QueueDriverResponse, "", false, false, false, false, nil)
		if err != nil {
			r.l.Error(ctx, "consume failed", err)
			time.Sleep(2 * time.Second)
			continue
		}

		r.l.Info(ctx, "start consuming ride requests", "queue", queue)

		// Цикл чтения сообщений
	consumeLoop:
		for {
			select {
			case <-ctx.Done():
				r.l.Info(ctx, "ride request consumer shutting down")
				return nil

			case msg, ok := <-msgs:
				if !ok {
					r.l.Warn(ctx, "message channel closed, reconnecting...")
					time.Sleep(2 * time.Second)
					continue consumeLoop
				}

				go func(d amqp091.Delivery) {
					var req models.DriverMatchResponse
					if err := json.Unmarshal(d.Body, &req); err != nil {
						r.l.Error(ctx, "failed to unmarshal driver match response", err)
						d.Nack(false, false) // не подтверждаем сообщение
						return
					}

					// добавляем в контекст переменные для логирования и трассировки
					ctx = wrap.WithRequestID(wrap.WithRideID(ctx, req.RideID.String()), d.CorrelationId)

					if err := handler(ctx, req); err != nil {
						r.l.Error(wrap.ErrorCtx(ctx, err), "failed to handle driver response", err)

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

type LocationUpdateHandler func(ctx context.Context, req models.RideLocationUpdate) error

func (r *RideMsgBroker) ConsumeDriverLocationUpdate(ctx context.Context, handler LocationUpdateHandler) error {
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

		const queue = "location_topic"
		msgs, err := r.client.Channel.Consume(r.QueueLocationUpdate, "", false, false, false, false, nil)
		if err != nil {
			r.l.Error(ctx, "consume failed", err)
			time.Sleep(2 * time.Second)
			continue
		}

		r.l.Info(ctx, "start consuming driver location updates", "queue", queue)

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

					ctx = wrap.WithRequestID(wrap.WithRideID(ctx, req.RideID.String()), d.CorrelationId)

					if err := handler(ctx, req); err != nil {
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
