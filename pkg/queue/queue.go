package queue

import (
	"github.com/lime-labs/metalcore/internal/pkg/common"
	"github.com/rs/zerolog/log"
	"github.com/streadway/amqp"
)

// Message struct creates...
type Message struct {
	MessageID     string
	SessionID     string
	CorrelationID string
	Queue         string
	ReplyTo       string
	Payload       []byte
}

// SendMessageToQueueChannel ...
func SendMessageToQueueChannel(channel *amqp.Channel, message Message) {
	err := channel.Publish(
		"",            // exchange
		message.Queue, // routing key
		false,         // mandatory
		false,         // immediate
		amqp.Publishing{
			ContentType:   "text/plain",
			MessageId:     message.MessageID,
			Body:          message.Payload,
			ReplyTo:       message.ReplyTo,
			CorrelationId: message.CorrelationID,
			AppId:         message.SessionID,
		})
	if err != nil {
		log.Error().Err(err).Str("component", "AMQP").Msg("error during publishing of message") // don't fataly fail, just log the error and continue
	}
}

// StartBackgroundPublisher ...
func StartBackgroundPublisher(messageBuffer <-chan Message, channel *amqp.Channel) {
	for message := range messageBuffer { // endlessly listen for messages that need to be published
		SendMessageToQueueChannel(channel, message)
	}
}

// CreateConnectionChannel returns an AMQP channel
func CreateConnectionChannel(AMQPconnectionString string, prefetch int) *amqp.Channel {
	conn, err := amqp.Dial(AMQPconnectionString)
	common.FailOnError(err, "failed to connect to RabbitMQ", "AMQP")
	//defer conn.Close()

	ch, err := conn.Channel()
	common.FailOnError(err, "failed to open a channel", "AMQP")
	//defer ch.Close()

	log.Info().Str("component", "AMQP").Msgf("successfully connected to AMQP broker at %v", AMQPconnectionString)

	if prefetch != 0 {
		err = ch.Qos(prefetch, 0, false)
		common.FailOnError(err, "failed to set QoS prefetch level on queue", "AMQP")
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
	common.FailOnError(err, "failed to declare a queue", "AMQP")

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
	common.FailOnError(err, "failed to consume messages from queue "+queueName, "AMQP")

	return msgs
}
