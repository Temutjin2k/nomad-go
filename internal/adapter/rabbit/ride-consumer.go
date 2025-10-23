package rabbit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/logger"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/rabbit"
	"github.com/rabbitmq/amqp091-go"
)

type RideConsumer struct {
	client *rabbit.RabbitMQ
	l      logger.Logger
}

func NewRideConsumer(client *rabbit.RabbitMQ, l logger.Logger) *RideConsumer {
	return &RideConsumer{
		client: client,
		l:      l,
	}
}

// helper для объявления и биндинга очереди
func (r *RideConsumer) declareAndBindQueue(ctx context.Context, queueName, bindingKey, exchangeName string) (amqp091.Queue, error) {
	const op = "RideConsumer.declareAndBindQueue"

	q, err := r.client.Channel.QueueDeclare(
		queueName,
		true, // durable
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ctx = wrap.WithAction(ctx, "declare_queue")
		return q, wrap.Error(ctx, fmt.Errorf("%s: failed to declare queue: %w", op, err))
	}

	if err := r.client.Channel.QueueBind(
		q.Name,
		bindingKey,
		exchangeName,
		false,
		nil,
	); err != nil {
		ctx = wrap.WithAction(ctx, "bind_queue")
		return q, wrap.Error(ctx, fmt.Errorf("%s: failed to bind queue: %w", op, err))
	}

	return q, nil
}

type HandlerFunc func(ctx context.Context, req models.RideRequestedMessage) error

func (r *RideConsumer) ConsumeRideRequest(ctx context.Context, fn HandlerFunc) error {
	const op = "RideConsumer.ConsumeRideRequest"

	if err := r.client.Channel.ExchangeDeclare("ride_topic", "topic", true, false, false, false, nil); err != nil {
		r.l.Error(ctx, "failed to declare exchange", err, "topic", "ride_topic")
	}

	for {
		// если контекст отменён — выходим
		if ctx.Err() != nil {
			r.l.Debug(ctx, "closing consume ride request by (ctx.Err)")
			return fmt.Errorf("%s: graceful shutdown", op)
		}

		// Подключение и биндинг очереди
		q, err := r.declareAndBindQueue(ctx, "ride_requests", "ride.request.*", "ride_topic")
		if err != nil {
			r.l.Error(ctx, "declare queue failed", err, "op", op)
			time.Sleep(2 * time.Second)
			continue // повторяем попытку
		}

		// Подписка на очередь
		msgs, err := r.client.Channel.Consume(
			q.Name,
			"",
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			r.l.Error(ctx, "consume failed", err, "op", op)
			time.Sleep(2 * time.Second)
			continue
		}
		r.l.Debug(ctx, "start consuming messages", "op", op)

		// Основной цикл чтения
	consumeLoop:
		for {
			select {
			case <-ctx.Done():
				r.l.Info(ctx, "ride request consume close (ctx Done)", "op", op)
				return nil

			case msg, ok := <-msgs:
				if !ok {
					r.l.Warn(ctx, "ride request channel closed, reconnecting", "op", op)
					r.client.Reconnect(ctx)
					time.Sleep(2 * time.Second)
					break consumeLoop
				}

				r.l.Debug(ctx, fmt.Sprintf("%s: received message — %s", op, string(msg.Body)))

				// обработка бизнес-логики
				go func() {

					var req models.RideRequestedMessage
					if err := json.Unmarshal(msg.Body, &req); err != nil {
						r.l.Error(ctx, "failed to decode ride request: %w", err, "op", op)
						return
					}

					if err := fn(ctx, req); err != nil {
						r.l.Error(ctx, "handler failed", err, "op", op)
					}
				}()
			}
		}

		// после break consumeLoop мы снова начнём с первой итерации for, пересоздав соединение и msgs
	}
}

func (r *RideConsumer) ConsumeMatchConfirmation(ctx context.Context) error {
	const op = "RideConsumer.ConsumeMatchConfirmation"
	return nil
}
