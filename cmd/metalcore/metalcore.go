package main

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/lime-labs/metalcore/internal/pkg/common"
	"github.com/lime-labs/metalcore/pkg/queue"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/streadway/amqp"
	"go.uber.org/automaxprocs/maxprocs" // currently needed since cgroup CPU limits are not mapped to GOMAXPROCS automatically without it
)

var config struct {
	ncpuLimit            int
	errorLimit           int
	amqpConnectionString string
	taskQueueName        string
	prefetchCount        int
	resultBufferSize     int
	servicePath          string
}

func startSubprocess(pathToBinary string, socket string, subprocessCounter chan<- int, errorCounter chan<- int) {
	log.Debug().Str("component", "IMP").Msgf("starting sub-process %v with socket %v", pathToBinary, socket)

	cmd := exec.Command(pathToBinary)

	cmd.Env = os.Environ() // provide env variables from parent process to sub-process
	cmd.Env = append(cmd.Env, "METALCORESOCKET="+socket)

	cmd.Stdout = os.Stdout // pass stdout...
	cmd.Stderr = os.Stderr // ...and stderr to parent process

	err := cmd.Start()
	if err != nil {
		log.Error().Err(err).Str("component", "IMP").Msgf("error starting sub-process with socket %v", socket)
		errorCounter <- 1 // increase error counter channel by 1
	} else {
		subprocessCounter <- 1 // increase sub-process counter channel by 1
		log.Debug().Str("component", "IMP").Msgf("successfully started sub-process %v with socket %v", pathToBinary, socket)
		log.Debug().Str("component", "IMP").Msg("waiting for sub-process to finish / exit...")
		err = cmd.Wait() // this blocks until the process exits (gracefully or not)
		if err != nil {
			log.Error().Err(err).Str("component", "IMP").Msg("sub-process exited with non-zero return code")
			errorCounter <- 1 // increase error counter channel by 1
		}
		subprocessCounter <- -1 // decrease sub-process counter channel by 1
	}
}

func handleRequest(socketConnection net.Conn, sessionID string, messageID string, task []byte, resultsBuffer chan<- queue.Message, resultQueue string, resultChannel *amqp.Channel) bool {
	log.Trace().Str("component", "IMP").Msgf("task messageID: %v | payload: %v", messageID, string(task))
	buf := make([]byte, 1024) // buffer to hold incoming data

	socketConnection.Write(task) // Send a task back to the process that established the connection

	len, err := socketConnection.Read(buf)
	if err != nil {
		log.Error().Err(err).Str("component", "IMP").Msg("received an error during sub-process task execution while reading task result from socket")
		return false
	}

	resultPayload := buf[:len]

	log.Trace().Str("component", "IMP").Msgf("result correlationID: %v | payload: %v", messageID, string(resultPayload))

	// write result to result buffer go channel
	result := queue.Message{CorrelationID: messageID, SessionID: sessionID, Queue: resultQueue, Payload: resultPayload}
	resultsBuffer <- result

	// signal successful handover to the result queue.
	// WARNING!: at this stage it's handed over to the result BUFFER, not yet submitted to the actual queue
	// on the broker side! might want to reconsider doing the confirm for the ACK more async as well.
	return true
}

func getSocketPath(socketName string, number int) string {
	osTempDir := os.TempDir()
	name := socketName + strconv.Itoa(number) + ".sock"
	return filepath.Join(osTempDir, name)
}

func createSocketListener(messages <-chan amqp.Delivery, resultsBuffer chan<- queue.Message, socket string, resultChannel *amqp.Channel) {
	log.Debug().Str("component", "IMP").Msgf("creating UNIX domain socket at %s", socket)

	err := os.RemoveAll(socket)
	common.FailOnError(err, "failed to remove existing sockets", "IMP")

	l, err := net.Listen("unix", socket)
	common.FailOnError(err, "socket listen error", "IMP")

	go func() {
		defer l.Close()

		for {
			conn, err := l.Accept()
			common.FailOnError(err, "accept error", "IMP")

			log.Debug().Str("component", "IMP").Msgf("client sub-process connected via [%s] on socket %s", conn.RemoteAddr().Network(), socket)

			for msg := range messages {
				// if the handler returns true then ACK, else NACK
				// to submit the message back into the rabbit queue for another round of processing
				if handleRequest(conn, msg.AppId, msg.MessageId, msg.Body, resultsBuffer, msg.ReplyTo, resultChannel) {
					msg.Ack(false) // true = ack all previously unacknowledged messages (batching of acks, go routine / thread fuckups ahead!),
				} else {
					msg.Nack(false, true)
				}
			}
			log.Fatal().Str("component", "IMP").Msg("message queue consumer closed - critical error, exiting now...")
		}
	}()
}

func subprocessCount(channel <-chan int) {
	var subprocessCount int
	for value := range channel { // endlessly listen for values to be added / subtracted from the count
		log.Trace().Str("component", "IMP").Msgf("value of sub-process count increase/decrease: %d", value)
		subprocessCount += value
		if subprocessCount == config.ncpuLimit {
			log.Info().Str("component", "IMP").Msgf("reached target of %d started worker instances of service %v!", config.ncpuLimit, config.servicePath)
		}
	}
}

func errorCount(channel <-chan int) {
	var errorCount int
	for value := range channel { // endlessly listen for values to be added / subtracted from the count
		log.Trace().Str("component", "IMP").Msgf("value of sub-process error count increase/decrease: %d", value)
		errorCount += value
		if errorCount == config.errorLimit {
			log.Fatal().Str("component", "IMP").Msgf("FATAL: reached the limit of %d failed starts or runs of service instances, exiting now...", config.errorLimit)
		}
	}
}

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	switch loglevel := os.Getenv("LOGLEVEL"); loglevel {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	if os.Getenv("LOGPRETTYPRINT") == "on" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	ncpu := runtime.NumCPU()
	log.Debug().Str("component", "IMP").Msgf("starting on a host with %d logical CPUs", ncpu)

	manualMaxCPULimit := os.Getenv("CPULIMIT")
	if manualMaxCPULimit != "" {
		var err error
		config.ncpuLimit, err = strconv.Atoi(manualMaxCPULimit)
		common.FailOnError(err, "error parsing CPULIMIT into an int", "IMP")
		runtime.GOMAXPROCS(config.ncpuLimit)
		log.Debug().Str("component", "IMP").Msgf("maxprocs: Updating GOMAXPROCS=%v: determined from env variable CPULIMIT", config.ncpuLimit)
	} else {
		maxprocs.Set(maxprocs.Logger(log.Debug().Str("component", "IMP").Msgf)) // read cgroup limits from runtime and set GOMAXPROCS accordingly
		config.ncpuLimit = runtime.GOMAXPROCS(0)                                // leave limit untouched, set value to current setting of GOMAXPROCS
	}
	log.Info().Str("component", "IMP").Msgf("number of usable logical CPUs for this IMP: %d", config.ncpuLimit)

	prefetchMultiplier := 100 // default value that can be overwritten with the env variable below
	if prefetchMultiplierString := os.Getenv("PREFETCHMULTIPLIER"); prefetchMultiplierString != "" {
		var err error
		prefetchMultiplier, err = strconv.Atoi(prefetchMultiplierString)
		common.FailOnError(err, "error parsing PREFETCHMULTIPLIER into an int", "IMP")
		log.Info().Str("component", "IMP").Msgf("prefetch multiplier has been manually set to %d via env variable PREFETCHMULTIPLIER", prefetchMultiplier)
	}

	config.prefetchCount = config.ncpuLimit * prefetchMultiplier
	config.resultBufferSize = config.prefetchCount

	log.Debug().Str("component", "IMP").Msgf("prefetch count for task queue set to %d messages", config.prefetchCount)
	log.Debug().Str("component", "IMP").Msgf("buffer size for results in internal memory set to %d items", config.resultBufferSize)

	config.errorLimit = 10 // default value that can be overwritten with the env variable below
	if errorLimitString := os.Getenv("ERRORLIMIT"); errorLimitString != "" {
		var err error
		config.errorLimit, err = strconv.Atoi(errorLimitString)
		common.FailOnError(err, "error parsing ERRORLIMIT into an int", "IMP")
		log.Info().Str("component", "IMP").Msgf("error limit has been manually set to %d via env variable ERRORLIMIT", config.errorLimit)
	}
	log.Debug().Str("component", "IMP").Msgf("error limit is set to %d", config.errorLimit)

	amqpHost := os.Getenv("AMQPHOST")
	amqpPort := os.Getenv("AMQPPORT")
	amqpUser := os.Getenv("AMQPUSER")
	amqpPassword := os.Getenv("AMQPPASSWORD")
	config.taskQueueName = os.Getenv("TASKQUEUENAME")
	config.servicePath = os.Getenv("SERVICEPATH")

	if amqpHost == "" || amqpUser == "" || amqpPassword == "" || config.taskQueueName == "" || config.servicePath == "" {
		log.Fatal().Str("component", "IMP").Msg("not all required env variables for an AMQP connection set, exiting now...")
	}

	config.amqpConnectionString = "amqp://" + amqpUser + ":" + amqpPassword + "@" + amqpHost + ":" + amqpPort
}

func main() {
	taskChannel := queue.CreateConnectionChannel(config.amqpConnectionString, config.prefetchCount)
	taskQueue := queue.DeclareQueue(taskChannel, config.taskQueueName)

	resultChannel := queue.CreateConnectionChannel(config.amqpConnectionString, 0)

	msgs := queue.ConsumeOnChannel(taskChannel, taskQueue.Name)
	// create a buffer queue / go channel to handle all result submissions in a dedicated, central go routine
	resultsBuffer := make(chan queue.Message, config.prefetchCount) // go channels are lmited to 64 KiB item size, be aware! An unbuffered since publishings are async by default, but this decouples it even more
	go queue.StartBackgroundPublisher(resultsBuffer, resultChannel)

	subprocessCounter := make(chan int)   // go channel to count started subprocesses
	go subprocessCount(subprocessCounter) // start the goroutine that syncs the count
	errorCounter := make(chan int)        // go channel to count errors happening during sub-process execution
	go errorCount(errorCounter)           // start the goroutine that syncs the error count

	// create a go routine for each concurrent thread requested
	log.Info().Str("component", "IMP").Msgf("starting %d worker instances of service %v...", config.ncpuLimit, config.servicePath)
	for i := 0; i < config.ncpuLimit; i++ {
		log.Debug().Str("component", "IMP").Msgf("creating local socket and starting worker for thread %d...", i)
		socket := getSocketPath("metalcoreTaskSocket", i)
		createSocketListener(msgs, resultsBuffer, socket, resultChannel)
		// start actual worker binary process here, pass and read parameters/result via socket created above
		go startSubprocess(config.servicePath, socket, subprocessCounter, errorCounter) // TODO: handle sub-process restarts on errors (use the go channel!)
	}

	forever := make(chan bool)
	<-forever // never end, inifitenly wait for all subprocesses
}
