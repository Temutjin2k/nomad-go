package rabbit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/rabbit"
	"github.com/rabbitmq/amqp091-go"
)

type DriverMsgBroker struct {
	broker   *rabbit.RabbitMQ
	exchange string
}

func NewDriverMsgBroker(broker *rabbit.RabbitMQ) *DriverMsgBroker {
	return &DriverMsgBroker{
		broker:   broker,
		exchange: "driver_topic",
	}
}

/*================= Publisher =====================*/

// PublishDriverStatus — публикует сообщение о статусе водителя
func (r *DriverMsgBroker) PublishDriverStatus(ctx context.Context, msg models.DriverStatusUpdateMessage) error {
	const op = "DriverMsgBroker.PublishDriverStatus"

	body, err := json.Marshal(msg)
	if err != nil {
		ctx = wrap.WithAction(ctx, "marshal_driver")
		return wrap.Error(ctx, fmt.Errorf("%s: failed to marshal message: %w", op, err))
	}

	key := fmt.Sprintf("driver.status.%s", msg.DriverID)

	if err := r.broker.Channel.PublishWithContext(
		ctx,
		r.exchange, // exchange
		key,        // routing key
		false,      // mandatory
		false,      // immediate
		amqp091.Publishing{
			ContentType: "application/json",
			Body:        body,
			Timestamp:   time.Now(),
		},
	); err != nil {
		ctx = wrap.WithAction(ctx, "publish_message")
		return wrap.Error(ctx, fmt.Errorf("%s: failed to publish with context: %w", op, err))
	}

	return nil
}

/*================= Consumer =====================*/

// ConsumeDriverStatus — слушает обновления статуса водителей
func (r *DriverMsgBroker) ConsumeDriverStatus(ctx context.Context, queueName, bindingKey string, handler func(context.Context, models.DriverStatusUpdateMessage) error) error {
	const op = "DriverMsgBroker.ConsumeDriverStatus"

	// Объявляем очередь
	q, err := r.broker.Channel.QueueDeclare(
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
	if err := r.broker.Channel.QueueBind(
		q.Name,
		bindingKey,
		r.exchange,
		false,
		nil,
	); err != nil {
		ctx = wrap.WithAction(ctx, "bind_queue")
		return wrap.Error(ctx, fmt.Errorf("%s: failed to bind queue: %w", op, err))
	}

	// Начинаем приём сообщений
	msgs, err := r.broker.Channel.Consume(
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
