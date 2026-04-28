package telemetry

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
)

// AMQPHeaderCarrier adapts amqp.Table to propagation.TextMapCarrier.
type AMQPHeaderCarrier amqp.Table

func (c AMQPHeaderCarrier) Get(key string) string {
	v, ok := c[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func (c AMQPHeaderCarrier) Set(key string, value string) {
	c[key] = value
}

func (c AMQPHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// InjectAMQP serializes the trace context from ctx into an amqp.Table
// to be set as Headers on amqp.Publishing.
func InjectAMQP(ctx context.Context) amqp.Table {
	headers := make(amqp.Table)
	otel.GetTextMapPropagator().Inject(ctx, AMQPHeaderCarrier(headers))
	return headers
}

// ExtractAMQP restores the trace context from an amqp.Delivery's Headers.
// Use the returned context as the parent for spans processing that delivery.
func ExtractAMQP(ctx context.Context, delivery *amqp.Delivery) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, AMQPHeaderCarrier(delivery.Headers))
}
