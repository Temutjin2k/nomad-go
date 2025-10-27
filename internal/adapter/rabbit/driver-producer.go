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

const (
	driverTopic    = "driver_topic"
	locationFanout = "location_fanout"
)

type DriverProducer struct {
	client    *rabbit.RabbitMQ
	exchanges map[string]string
}

func NewDriverProducer(client *rabbit.RabbitMQ) *DriverProducer {
	p := &DriverProducer{
		client: client,
		exchanges: map[string]string{
			driverTopic:    "topic",
			locationFanout: "fanout",
		},
	}
	return p
}

func (r *DriverProducer) publish(ctx context.Context, exchange, routingKey string, msg any) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if err := r.client.Channel.ExchangeDeclare(exchange, r.exchanges[exchange], true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}

	pub := amqp091.Publishing{
		ContentType: "application/json",
		Body:        body,
		Timestamp:   time.Now(),
	}

	if err := retry(5, time.Second*2,
		func() error {
			return r.client.Channel.PublishWithContext(ctx, exchange, routingKey, false, false, pub)
		}); err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	return nil
}

func retry(n int, sleep time.Duration, fn func() error) error {
	var err error
	for i := 0; i < n; i++ {
		if err = fn(); err == nil {
			return nil
		}
		time.Sleep(sleep)
	}
	return err
}

func (r *DriverProducer) PublishDriverStatus(ctx context.Context, msg models.DriverStatusUpdateMessage) error {
	ctx = wrap.WithAction(ctx, "publish_driver_status")
	key := fmt.Sprintf("driver.status.%s", msg.DriverID)

	if err := r.publish(ctx, driverTopic, key, msg); err != nil {
		return wrap.Error(ctx, err)
	}
	return nil
}

func (r *DriverProducer) PublishDriverResponse(ctx context.Context, msg models.DriverMatchResponse) error {
	ctx = wrap.WithAction(ctx, "publish_driver_response")
	key := fmt.Sprintf("driver.response.%s", msg.RideID)

	if err := r.publish(ctx, driverTopic, key, msg); err != nil {
		return wrap.Error(ctx, err)
	}
	return nil
}

func (r *DriverProducer) PublishLocationUpdate(ctx context.Context, msg models.RideLocationUpdate) error {
	ctx = wrap.WithAction(ctx, "publish_location_update")
	key := "" // N/A

	if err := r.publish(ctx, driverTopic, key, msg); err != nil {
		return wrap.Error(ctx, err)
	}
	return nil
}
