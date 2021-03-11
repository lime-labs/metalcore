package queue

import (
	"log"

	"github.com/lime-labs/metalcore/internal/pkg/common"
	"github.com/streadway/amqp"
)

// Result creates...
type Result struct {
	MessageID   string
	SessionID   string
	ResultQueue string
	Payload     []byte
}

// StartBackgroundResultsPublisher provides...
func StartBackgroundResultsPublisher(resultsBuffer <-chan Result, resultChannel *amqp.Channel) {
	// endlessly listen for results that need to be published
	for result := range resultsBuffer {
		//result := <-resultsBuffer // synchronization

		err := resultChannel.Publish(
			"",                 // exchange
			result.ResultQueue, // routing key
			false,              // mandatory
			false,              // immediate
			amqp.Publishing{
				ContentType:   "text/plain",
				Body:          result.Payload,
				CorrelationId: result.MessageID,
				AppId:         result.SessionID,
			})
		if err != nil {
			log.Println("[AMQP]     error during publishiung of result", err)
		}
	}
}

// CreateConnectionChannel returns an AMQP channel
func CreateConnectionChannel(AMQPconnectionString string, prefetch int) *amqp.Channel {
	conn, err := amqp.Dial(AMQPconnectionString)
	common.FailOnError(err, "[AMQP]     failed to connect to RabbitMQ")
	//defer conn.Close()

	ch, err := conn.Channel()
	common.FailOnError(err, "[AMQP]     failed to open a channel")
	//defer ch.Close()

	log.Printf("[AMQP]     successfully connected to AMQP broker: " + AMQPconnectionString)

	if prefetch != 0 {
		err = ch.Qos(prefetch, 0, false)
		common.FailOnError(err, "[AMQP]     failed to set QoS prefetch level on queue")
	}

	return ch
}

// DeclareQueue creates a queue via a given channel, if it doesn't already exist
func DeclareQueue(channel *amqp.Channel, queueName string) amqp.Queue {
	q, err := channel.QueueDeclare(
		queueName, // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	common.FailOnError(err, "[AMQP]     failed to declare a queue")

	return q
}

// ConsumeOnChannel return a message go channel which holds the messages from the given AMQP channel and queue
func ConsumeOnChannel(channel *amqp.Channel, queueName string) <-chan amqp.Delivery {
	msgs, err := channel.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	common.FailOnError(err, "[AMQP]     failed to consume messages from queue "+queueName)

	return msgs
}
