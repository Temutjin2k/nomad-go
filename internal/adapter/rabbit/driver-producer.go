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
	client   *rabbit.RabbitMQ
	exchange string
}

func NewDriverProducer(client *rabbit.RabbitMQ) *DriverProducer {
	return &DriverProducer{
		client:   client,
		exchange: "driver_topic",
	}
}

// initExchange — гарантирует наличие exchange
func (r *DriverProducer) initExchange(_ context.Context) error {
	return r.client.Channel.ExchangeDeclare(
		r.exchange,
		"topic",
		true,  // durable
		false, // autoDelete
		false, // internal
		false, // noWait
		nil,   // args
	)
}

func (r *DriverProducer) publish(ctx context.Context, routingKey string, msg any) error {
	const op = "DriverProducer.publish"

	body, err := json.Marshal(msg)
	if err != nil {
		return wrap.Error(ctx, fmt.Errorf("%s: marshal: %w", op, err))
	}

	// гарантируем, что exchange существует
	if err := r.initExchange(ctx); err != nil {
		return wrap.Error(ctx, fmt.Errorf("%s: declare exchange: %w", op, err))
	}

	pub := amqp091.Publishing{
		ContentType: "application/json",
		Body:        body,
		Timestamp:   time.Now(),
	}

	if err := r.client.Channel.PublishWithContext(ctx, r.exchange, routingKey, false, false, pub); err != nil {
		return wrap.Error(ctx, fmt.Errorf("%s: publish: %w", op, err))
	}
	return nil
}

func (r *DriverProducer) PublishDriverStatus(ctx context.Context, msg models.DriverStatusUpdateMessage) error {
	key := fmt.Sprintf("driver.status.%s", msg.DriverID)
	return r.publish(ctx, key, msg)
}

func (r *DriverProducer) PublishRideStatus(ctx context.Context, msg models.RideStatusUpdateMessage) error {
	key := fmt.Sprintf("ride.status.%s", msg.RideID)
	return r.publish(ctx, key, msg)
}

func (r *DriverProducer) PublishDriverResponse(ctx context.Context, msg models.DriverMatchResponse) error {
	key := fmt.Sprintf("driver.response.%s", msg.RideID)
	return r.publish(ctx, key, msg)
}
