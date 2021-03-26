package main

import (
	"crypto/rand"
	"flag"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/beanstalkd/go-beanstalk"
	"github.com/lime-labs/metalcore/internal/pkg/common"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func throughputCount(channel <-chan float64, threads int, purpose string, wg *sync.WaitGroup) {
	defer wg.Done()
	var total float64
	count := 0
	for value := range channel { // endlessly listen for values to be added / subtracted from the count
		log.Trace().Str("component", "client").Msgf("received value of %v throughput increase/decrease: %.2f", purpose, value)
		total += value
		count++
		if count == threads {
			log.Info().Str("component", "client").Msgf("total %v throughput using %d threads: %.2f tasks/sec", purpose, threads, total)
			break
		}
	}
}

func main() {
	clientStartTime := time.Now()

	// commandline flags
	sleepPtr := flag.Int("sleep", 100, "number of ms to sleep for on process worker thread side")
	taskNumPtr := flag.Int("tasks", 10000, "number of tasks to send to the queue")
	sizePtr := flag.Int("size", 1024, "number of bytes to use as fake payload")
	parallelPtr := flag.Int("parallel", 1, "number of threads and TCP connections for task submission and retrieval to spawn")
	taskQueuePtr := flag.String("taskqueue", "tasks", "name of the task queue to use")
	resultQueuePtr := flag.String("resultqueue", "results", "name of the result queue to use")
	queueServerPtr := flag.String("server", "127.0.0.1:11300", "Hostname / FQDN / IP and port of queue server to use")
	//sessionPtr := flag.String("session", "metalcore-client-app/123456789", "name of the session identifier to use")
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

	//hostname, err := os.Hostname()
	//common.FailOnError(err, "failed to get hostname", "client")

	taskQueueName := *taskQueuePtr
	resultQueueName := *resultQueuePtr

	sleepDuration := strconv.Itoa(*sleepPtr) // sleep task for xxx ms

	tasksPerThread := *taskNumPtr / *parallelPtr
	remainder := *taskNumPtr % *parallelPtr

	if remainder != 0 {
		log.Warn().Str("component", "client").Msgf("WARNING! Using %d threads for submitting %d tasks leaves a remainder of %d task(s) that will be skipped!", *parallelPtr, *taskNumPtr, remainder)
	}
	log.Info().Str("component", "client").Msgf("starting %d threads for submitting %d tasks and %d threads for retrieving the results, please stand by...", *parallelPtr, *taskNumPtr-remainder, *parallelPtr)

	payload := make([]byte, *sizePtr)
	rand.Read(payload)
	log.Info().Str("component", "client").Msgf("fake payload size set to %d bytes of random data", *sizePtr)

	var wg sync.WaitGroup
	wg.Add(2)
	submissionThroughputCounter := make(chan float64)                                     // go channel to count submission throughput
	go throughputCount(submissionThroughputCounter, *parallelPtr, "task submission", &wg) // start the goroutine that syncs the count
	resultThroughputCounter := make(chan float64)                                         // go channel to count result throughput
	go throughputCount(resultThroughputCounter, *parallelPtr, "result retrieval", &wg)    // start the goroutine that syncs the count

	for p := 0; p < *parallelPtr; p++ {
		wg.Add(1)
		go func(counter int) { // publish in one thread
			defer wg.Done()

			taskConnection, err := beanstalk.Dial("tcp", *queueServerPtr)
			common.FailOnError(err, "failed to connect to work queue server", "client")
			taskTube := beanstalk.NewTube(taskConnection, taskQueueName)

			log.Debug().Str("component", "client").Msgf("starting publishing thread %d on client, submitting %d tasks to task queue %s...", counter, tasksPerThread, taskQueueName)

			start := time.Now()
			for j := 0; j < tasksPerThread; j++ {
				id, err := taskTube.Put(payload, 1, 0, 5*time.Second) // publish task with random fake payload
				common.LogOnError(err, "error putting task on task queue", "client")
				log.Trace().Str("component", "client").Msgf("submitted task #: %v, task payload: %v", id, sleepDuration)
			}
			putDuration := time.Since(start)
			log.Debug().Str("component", "client").Msgf("thread %d DONE publishing tasks to task queue, total runtime: %v", counter, putDuration.String())

			throughput := float64(*taskNumPtr) / float64(putDuration) * float64(time.Second)
			log.Debug().Str("component", "client").Msgf("task submission rate of thread %d: %.2f tasks/sec", counter, throughput)
			submissionThroughputCounter <- throughput
		}(p)
	}

	for p := 0; p < *parallelPtr; p++ {
		wg.Add(1)
		go func(counter int) { // consume results in another thread
			defer wg.Done()

			resultConnection, err := beanstalk.Dial("tcp", *queueServerPtr)
			common.FailOnError(err, "failed to connect to work queue server", "client")
			resultTubeSet := beanstalk.NewTubeSet(resultConnection, resultQueueName)

			log.Debug().Str("component", "client").Msgf("starting result retrieval thread %d on client, receiving results from queue %s...", counter, resultQueueName)

			start := time.Now()
			for i := 0; i < tasksPerThread; i++ {
				id, body, err := resultTubeSet.Reserve(10 * time.Hour) // get task
				common.LogOnError(err, "error getting result from result queue", "client")
				log.Trace().Str("component", "client").Msgf("Result #: %v, result payload: %v", id, string(body))

				err = resultConnection.Delete(id)
				common.LogOnError(err, "error deleting result on queue", "client")
				log.Trace().Str("component", "client").Msgf("succesfully deleted result #: %v", id)
			}
			getDuration := time.Since(start)
			log.Debug().Str("component", "client").Msgf("thread %d DONE retrieving %d results from result queue, total runtime: %v", counter, tasksPerThread, getDuration.String())

			throughput := float64(*taskNumPtr) / float64(getDuration) * float64(time.Second)
			log.Debug().Str("component", "client").Msgf("result retrieval rate of thread %d: %.2f tasks/sec", counter, throughput)
			resultThroughputCounter <- throughput
		}(p)
	}

	wg.Wait() // wait / sync for threads / goroutines
	log.Info().Str("component", "client").Msgf("total client runtime: %v", time.Since(clientStartTime))
}
