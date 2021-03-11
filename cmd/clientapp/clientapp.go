package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/lime-labs/metalcore/internal/pkg/common"
	"github.com/lime-labs/metalcore/pkg/queue"
	"github.com/streadway/amqp"
)

func main() {
	// commandline flags
	sleepPtr := flag.Int("sleep", 100, "number of ms to sleep for on process worker thread side")
	taskNumPtr := flag.Int("tasks", 10000, "number of tasks to send to the queue")
	prefetchCountPtr := flag.Int("prefetch", 100, "prefetch this number of tasks from the result queue")
	amqpPtr := flag.String("amqp", "amqp://guest:guest@localhost:5672", "AMQP connection string")
	taskQueuePtr := flag.String("queue", "metalcore-tasks", "name of the task queue to use")
	sessionPtr := flag.String("session", "metalcore-client-app/123456789", "name of the session identifier to use")
	quietModePtr := flag.Bool("quiet", false, "set this flag to disable any printing of results and speed up result consumption")
	flag.Parse()

	hostname, err := os.Hostname()
	common.FailOnError(err, "failed to get hostname")

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
	// publish in one thread
	go func() {
		defer wg.Done()

		log.Printf("Starting publishing thread on client, submitting %d tasks to task queue %s...", *taskNumPtr, taskQueueName)

		for j := 0; j < *taskNumPtr; j++ {
			err := taskChannel.Publish(
				"",             // exchange
				taskQueue.Name, // routing key
				false,          // mandatory
				false,          // immediate
				amqp.Publishing{
					ContentType: "text/plain",
					Body:        []byte(body),
					ReplyTo:     resultQueue.Name,
					AppId:       sessionID,
					MessageId:   strconv.Itoa(j),
				})
			common.FailOnError(err, "Failed to publish a message")
		}
		log.Println("DONE publishing tasks to task queue!")
	}()

	wg.Add(1)
	// consume results in another thread
	go func() {
		defer wg.Done()

		log.Printf("Starting result consuming thread on client, receiving results from queue %s...", resultQueueName)

		msgs := queue.ConsumeOnChannel(resultChannel, resultQueueName)

		i := 0
		for msg := range msgs {

			if !*quietModePtr {
				log.Println("Session #: " + msg.AppId + " received result. Task #: " + msg.CorrelationId + ", result: " + string(msg.Body))
			}
			msg.Ack(false) // despite the looks of this format, this actually ACKs the message

			i++
			if i == *taskNumPtr {
				log.Printf("DONE receiving results from result queue, received %d results.", *taskNumPtr)
			}
		}

	}()

	wg.Wait() // wait / sync for threads / goroutines

}