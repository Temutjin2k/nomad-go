package types

const (
	ActionRabbitMQConnected       = "rabbitmq_connected"
	ActionRabbitConnectionClosed  = "rabbitmq_connection_closed"
	ActionRabbitConnectionClosing = "rabbitmq_connection_closing"
	ActionRabbitReconnected       = "rabbitmq_reconnection_success"

	ActionDatabaseTransactionFailed = "database_transaction_failed"
	ActionExternalServiceFailed     = "external_service_failed"
)
