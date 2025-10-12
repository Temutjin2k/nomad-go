package rabbit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	wrap "github.com/Temutjin2k/ride-hail-system/pkg/logger/wrapper"
	"github.com/Temutjin2k/ride-hail-system/pkg/rabbit"
)

type RideMsgBroker struct {
	broker   *rabbit.RabbitMQ
	exchange string
}

func NewRideMsgBroker(broker *rabbit.RabbitMQ) *RideMsgBroker {
	return &RideMsgBroker{
		broker:   broker,
		exchange: "ride_topic", // Как указано в архитектуре
	}
}

// публикует событие о новой поездке для поиска водителя.
// Отправляет в exchange 'ride_topic' с ключом 'ride.request.{ride_type}'.
func (r *RideMsgBroker) PublishRideRequested(ctx context.Context, msg models.RideRequestedMessage) error {
	const op = "RideMsgBroker.PublishRideRequested"

	body, err := json.Marshal(msg)
	if err != nil {
		ctx = wrap.WithAction(ctx, "marshal_ride_request")
		return wrap.Error(ctx, fmt.Errorf("%s: failed to marshal message: %w", op, err))
	}

	// Ключ маршрутизации, например, "ride.request.ECONOMY"
	key := fmt.Sprintf("ride.request.%s", msg.RideType)

	if err := r.broker.Channel.PublishWithContext(
		ctx,
		r.exchange, // exchange
		key,        // routing key
		false,      // mandatory
		false,      // immediate
		amqp091.Publishing{
			ContentType:   "application/json",
			CorrelationId: msg.CorrelationID, // для трассировки
			Body:          body,
			Timestamp:     time.Now(),
		},
	); err != nil {
		ctx = wrap.WithAction(ctx, "publish_message")
		return wrap.Error(ctx, fmt.Errorf("%s: failed to publish with context: %w", op, err))
	}

	return nil
}

// публикует событие об изменении статуса поездки.
// Отправляет в exchange 'ride_topic' с ключом 'ride.status.{status}'.
func (r *RideMsgBroker) PublishRideStatus(ctx context.Context, msg models.RideStatusUpdateMessage) error {
	const op = "RideMsgBroker.PublishRideStatus"

	body, err := json.Marshal(msg)
	if err != nil {
		ctx = wrap.WithAction(ctx, "marshal_ride_status")
		return wrap.Error(ctx, fmt.Errorf("%s: failed to marshal message: %w", op, err))
	}

	key := fmt.Sprintf("ride.status.%s", msg.Status)

	if err := r.broker.Channel.PublishWithContext(
		ctx,
		r.exchange, // exchange
		key,        // routing key
		false,      // mandatory
		false,      // immediate
		amqp091.Publishing{
			ContentType:   "application/json",
			CorrelationId: msg.CorrelationID,
			Body:          body,
			Timestamp:     time.Now(),
		},
	); err != nil {
		ctx = wrap.WithAction(ctx, "publish_message")
		return wrap.Error(ctx, fmt.Errorf("%s: failed to publish with context: %w", op, err))
	}

	return nil
}