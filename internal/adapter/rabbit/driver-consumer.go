package rabbit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/rabbit"
)

type DriverConsumer struct {
	client *rabbit.RabbitMQ
}

func NewDriverConsumer(client *rabbit.RabbitMQ) *DriverConsumer {
	return &DriverConsumer{
		client: client,
	}
}

/*================= Consumer =====================*/

// ConsumeDriverStatus — слушает обновления статуса водителей
func (r *DriverConsumer) ConsumeDriverStatus(ctx context.Context, queueName, bindingKey string, handler func(context.Context, models.DriverStatusUpdateMessage) error) error {
	const op = "DriverProducer.ConsumeDriverStatus"

	// Объявляем очередь
	q, err := r.client.Channel.QueueDeclare(
		queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		ctx = wrap.WithAction(ctx, "declare_queue")
		return wrap.Error(ctx, fmt.Errorf("%s: failed to declare queue: %w", op, err))
	}

	// Привязываем очередь к exchange по ключу
	if err := r.client.Channel.QueueBind(
		q.Name,
		bindingKey,
		"driver_topic",
		false,
		nil,
	); err != nil {
		ctx = wrap.WithAction(ctx, "bind_queue")
		return wrap.Error(ctx, fmt.Errorf("%s: failed to bind queue: %w", op, err))
	}

	// Начинаем приём сообщений
	msgs, err := r.client.Channel.Consume(
		q.Name,
		"",
		true,  // auto-ack
		false, // exclusive
		false,
		false,
		nil,
	)
	if err != nil {
		ctx = wrap.WithAction(ctx, "consume data")
		return wrap.Error(ctx, fmt.Errorf("%s: failed to register consumer: %w", op, err))
	}

	// Запускаем обработку
	go func() {
		for d := range msgs {
			var message models.DriverStatusUpdateMessage
			if err := json.Unmarshal(d.Body, &message); err != nil {
				continue
			}

			handler(ctx, message)
		}
	}()

	return nil
}
