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

type DriverProducer struct {
	client *rabbit.RabbitMQ
}

func NewDriverProducer(client *rabbit.RabbitMQ) *DriverProducer {
	return &DriverProducer{
		client: client,
	}
}

// PublishDriverStatus — публикует сообщение о статусе водителя
func (r *DriverProducer) PublishDriverStatus(ctx context.Context, msg models.DriverStatusUpdateMessage) error {
	const op = "DriverProducer.PublishDriverStatus"

	body, err := json.Marshal(msg)
	if err != nil {
		ctx = wrap.WithAction(ctx, "marshal_driver")
		return wrap.Error(ctx, fmt.Errorf("%s: failed to marshal message: %w", op, err))
	}

	key := fmt.Sprintf("driver.status.%s", msg.DriverID)

	if err := r.client.Channel.PublishWithContext(
		ctx,
		"driver_topic", // exchange
		key,            // routing key
		false,          // mandatory
		false,          // immediate
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

func (r *DriverProducer) PublishRideStatus(ctx context.Context, msg models.RideStatusUpdateMessage) error {
	const op = "DriverProducer.PublishRideStatus"

	body, err := json.Marshal(msg)
	if err != nil {
		ctx = wrap.WithAction(ctx, "marshal_ride")
		return wrap.Error(ctx, fmt.Errorf("%s: failed to marshal message: %w", op, err))
	}
	key := fmt.Sprintf("ride.status.%s", msg.RideID)

	if err := r.client.Channel.PublishWithContext(
		ctx,
		"driver_topic", // exchange
		key,            // routing key
		false,          // mandatory
		false,          // immediate
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
