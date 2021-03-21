package main

import (
	"flag"
	"os"
	"strconv"
	"sync"

	"github.com/lime-labs/metalcore/internal/pkg/common"
	"github.com/lime-labs/metalcore/pkg/queue"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// commandline flags
	sleepPtr := flag.Int("sleep", 100, "number of ms to sleep for on process worker thread side")
	taskNumPtr := flag.Int("tasks", 10000, "number of tasks to send to the queue")
	prefetchCountPtr := flag.Int("prefetch", 100, "prefetch this number of tasks from the result queue")
	amqpPtr := flag.String("amqp", "amqp://guest:guest@localhost:5672", "AMQP connection string")
	taskQueuePtr := flag.String("queue", "metalcore-tasks", "name of the task queue to use")
	sessionPtr := flag.String("session", "metalcore-client-app/123456789", "name of the session identifier to use")
	logLevelPtr := flag.String("loglevel", "info", "set this to debug or trace to enable more verbose log outputs [slows down performance!]")
	prettyLogPtr := flag.Bool("pretty", false, "set this flag to enable human readable output of the log [otherwise JSON format will be used]")
	flag.Parse()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	switch *logLevelPtr {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	if *prettyLogPtr {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	hostname, err := os.Hostname()
	common.FailOnError(err, "failed to get hostname", "client")

	sessionID := *sessionPtr

	taskQueueName := *taskQueuePtr
	taskChannel := queue.CreateConnectionChannel(*amqpPtr, 0)
	taskQueue := queue.DeclareQueue(taskChannel, taskQueueName)

	resultQueueName := hostname
	resultChannel := queue.CreateConnectionChannel(*amqpPtr, *prefetchCountPtr)
	resultQueue := queue.DeclareQueue(resultChannel, resultQueueName)

	// sleep task for xxx ms
	body := strconv.Itoa(*sleepPtr)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() { // publish in one thread
		defer wg.Done()

		log.Info().Str("component", "client").Msgf("Starting publishing thread on client, submitting %d tasks to task queue %s...", *taskNumPtr, taskQueueName)

		for j := 0; j < *taskNumPtr; j++ {
			task := queue.Message{MessageID: strconv.Itoa(j), SessionID: sessionID, Queue: taskQueue.Name, ReplyTo: resultQueue.Name, Payload: []byte(body)}
			queue.SendMessageToQueueChannel(taskChannel, task)
			log.Trace().Str("component", "client").Msgf("Session #: %v sent task. Task #: %v, task payload: %v", task.SessionID, task.MessageID, string(task.Payload))
		}
		log.Info().Str("component", "client").Msg("DONE publishing tasks to task queue!")
	}()

	wg.Add(1)
	go func() { // consume results in another thread
		defer wg.Done()

		log.Info().Str("component", "client").Msgf("Starting result consuming thread on client, receiving results from queue %s...", resultQueueName)

		msgs := queue.ConsumeOnChannel(resultChannel, resultQueueName)

		i := 0
		for msg := range msgs {
			log.Trace().Str("component", "client").Msgf("Session #: %v received result. Task #: %v, result payload: %v", msg.AppId, msg.CorrelationId, string(msg.Body))
			msg.Ack(false) // despite the looks of this format, this actually ACKs the message

			if i++; i == *taskNumPtr {
				log.Info().Str("component", "client").Msgf("DONE receiving results from result queue, received %d results.", *taskNumPtr)
			}
		}
	}()

	wg.Wait() // wait / sync for threads / goroutines
}
