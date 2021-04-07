package main

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/beanstalkd/go-beanstalk"
	"github.com/lime-labs/metalcore/internal/pkg/common"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	api "github.com/lime-labs/metalcore/api/v1"
	"google.golang.org/protobuf/proto"

	"go.uber.org/automaxprocs/maxprocs" // currently needed since cgroup CPU limits are not mapped to GOMAXPROCS automatically without it
)

var config struct {
	ncpuLimit             int
	errorLimit            int
	queueConnectionString string
	taskQueueName         string
	resultQueueName       string
	connShareFactor       int
	servicePath           string
}

type queueConn struct {
	taskConn *beanstalk.Conn
	batches  *beanstalk.TubeSet
	results  *beanstalk.Tube
}

func startSubprocess(pathToBinary string, socket net.Listener, subprocessCounter chan<- int, errorCounter chan<- int) {
	log.Debug().Str("component", "IMP").Msgf("starting sub-process %v with socket %v", pathToBinary, socket.Addr().String())

	cmd := exec.Command(pathToBinary)

	cmd.Env = os.Environ() // provide env variables from parent process to sub-process
	cmd.Env = append(cmd.Env, "METALCORESOCKETTYPE="+socket.Addr().Network())
	cmd.Env = append(cmd.Env, "METALCORESOCKETADDR="+socket.Addr().String())

	cmd.Stdout = os.Stdout // pass stdout...
	cmd.Stderr = os.Stderr // ...and stderr to parent process

	err := cmd.Start()
	if err != nil {
		log.Error().Err(err).Str("component", "IMP").Msgf("error starting sub-process with socket %v", socket.Addr().String())
		errorCounter <- 1 // increase error counter channel by 1
	} else {
		subprocessCounter <- 1 // increase sub-process counter channel by 1
		log.Debug().Str("component", "IMP").Msgf("successfully started sub-process %v with socket %v", pathToBinary, socket.Addr().String())
		log.Debug().Str("component", "IMP").Msg("waiting for sub-process to finish / exit...")
		err = cmd.Wait() // this blocks until the process exits (gracefully or not)
		if err != nil {
			log.Error().Err(err).Str("component", "IMP").Msg("sub-process exited with non-zero return code")
			errorCounter <- 1 // increase error counter channel by 1
		}
		subprocessCounter <- -1 // decrease sub-process counter channel by 1
	}
}

func handleRequest(socketConnection net.Conn, task []byte, results *beanstalk.Tube) bool {
	common.Writer(socketConnection, task, "IMP")            // send task to sub-process
	resultPayload := common.Reader(socketConnection, "IMP") // receive result from sub-process

	id, err := results.Put(resultPayload, 1, 0, 120*time.Second) // send result to result queue
	if err != nil {
		common.LogOnError(err, "error putting result on result queue: "+results.Name, "IMP")
		return false
	}
	log.Trace().Str("component", "IMP").Msgf("sent result #: %d, result payload: %d bytes, result queue: %v ", id, len(resultPayload), results.Name)
	return true
}

func createSocket(number int) net.Listener {
	socketPath := filepath.Join(os.TempDir(), "metalcore"+strconv.Itoa(number)+".sock")

	err := os.RemoveAll(socketPath)
	common.FailOnError(err, "failed to remove existing sockets", "IMP")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Warn().Str("component", "IMP").Msg("local UNIX socket listen error, will fallback on local TCP with a random (free) port...")
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		common.FailOnError(err, "local TCP socket listen error, exiting now...", "IMP")
	}

	log.Debug().Str("component", "IMP").Msgf("created local IPC socket for thread #%d. Type: %v, address: %v", number, listener.Addr().Network(), listener.Addr().String())
	return listener
}

func startBatchConsumer(connCollection queueConn, socket net.Listener, errorCounter chan<- int) {
	queueConnection := connCollection.taskConn
	batches := connCollection.batches
	results := connCollection.results

	defer socket.Close()
	for {
		conn, err := socket.Accept()
		common.FailOnError(err, "accept error", "IMP")

		log.Debug().Str("component", "IMP").Msgf("client sub-process connected via [%s] on socket %s", conn.RemoteAddr().Network(), socket.Addr().String())

		for {
			id, batchFromQueue, err := batches.Reserve(1 * time.Hour) // get batch from queue
			if err != nil {
				if err == beanstalk.ErrDeadline { // TODO/FIXME: this does NOT correctly check for a deadline error!
					log.Warn().Str("component", "IMP").Msgf("WARNING! Thread using socket %v tried to reserve a batch from the queue but received DEADLINE SOON error, please handle it!", socket)
					// TODO: handle deadline soon errors properly!
				} else {
					log.Error().Err(err).Str("component", "IMP").Msg("error getting batch from task queue")
				}
				continue // skip rest of the loop, begin next cycle
			}
			log.Trace().Str("component", "IMP").Msgf("received batch #: %d, batch payload: %d bytes from batch queue(s)", id, len(batchFromQueue))

			batch := &api.Batch{}
			err = proto.Unmarshal(batchFromQueue, batch)
			common.LogOnError(err, "error while deserializing batch received from queue", "IMP")

			for _, task := range batch.Tasks {
				log.Trace().Str("component", "IMP").Msgf("task payload size sent to socket: %d bytes", len(task.Payload))

				if handleRequest(conn, task.Payload, results) { // if task was successfully handled and result properly submitted
					// TODO: do something meaningful here while still inside the batch loop (batch results together, handle errors, increase a success counter or something else)
				} else {
					// TODO: put batch/task in failed queue/tube
					errorCounter <- 1
				}
			} // batch done
			err = queueConnection.Delete(id)
			common.LogOnError(err, "error deleting batch from task queue", "IMP")
			log.Trace().Str("component", "IMP").Msgf("succesfully deleted batch #: %v", id)
		}
		//log.Fatal().Str("component", "IMP").Msg("batch queue consumer closed - critical error, exiting now...")
	}
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

func createQueueConnectionsAndTubes(connString string, batchQueueName string, resultQueueName string) (queueConnInstance queueConn) {
	var err error
	queueConnInstance.taskConn, err = beanstalk.Dial("tcp", connString)
	common.FailOnError(err, "failed to connect to work queue server", "IMP")

	resultConn, err := beanstalk.Dial("tcp", connString)
	common.FailOnError(err, "failed to connect to work queue server", "IMP")

	queueConnInstance.batches = beanstalk.NewTubeSet(queueConnInstance.taskConn, batchQueueName)
	queueConnInstance.results = beanstalk.NewTube(resultConn, resultQueueName)

	return
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

	config.errorLimit = 10 // default value that can be overwritten with the env variable below
	if errorLimitString := os.Getenv("ERRORLIMIT"); errorLimitString != "" {
		var err error
		config.errorLimit, err = strconv.Atoi(errorLimitString)
		common.FailOnError(err, "error parsing ERRORLIMIT into an int", "IMP")
		log.Info().Str("component", "IMP").Msgf("error limit has been manually set to %d via env variable ERRORLIMIT", config.errorLimit)
	}
	log.Debug().Str("component", "IMP").Msgf("error limit is set to %d", config.errorLimit)

	config.connShareFactor = 1 // default value that can be overwritten with the env variable below
	if connShareFactorString := os.Getenv("CONNSHAREFACTOR"); connShareFactorString != "" {
		var err error
		config.connShareFactor, err = strconv.Atoi(connShareFactorString)
		common.FailOnError(err, "error parsing CONNSHAREFACTOR into an int", "IMP")
		log.Info().Str("component", "IMP").Msgf("connection sharing factor has been manually set to %d via env variable CONNSHAREFACTOR", config.connShareFactor)
	}
	log.Debug().Str("component", "IMP").Msgf("connection sharing factor is set to %d", config.connShareFactor)

	queueHost := os.Getenv("QUEUEHOST")
	queuePort := os.Getenv("QUEUEPORT")

	config.taskQueueName = os.Getenv("TASKQUEUENAME")
	config.resultQueueName = os.Getenv("RESULTQUEUENAME")
	config.servicePath = os.Getenv("SERVICEPATH")

	if queueHost == "" || queuePort == "" || config.taskQueueName == "" || config.resultQueueName == "" || config.servicePath == "" {
		log.Fatal().Str("component", "IMP").Msg("not all required env variables for a queue connection set, exiting now...")
	}

	config.queueConnectionString = queueHost + ":" + queuePort
}

func main() {
	subprocessCounter := make(chan int)   // go channel to count started subprocesses
	go subprocessCount(subprocessCounter) // start the goroutine that syncs the count
	errorCounter := make(chan int)        // go channel to count errors happening during sub-process execution
	go errorCount(errorCounter)           // start the goroutine that syncs the error count

	var connections []queueConn // "collect" connection tuples to be referenced for sharing, if desired (default: 2 connections per thread, not shared)

	// create a go routine for each concurrent thread requested
	log.Info().Str("component", "IMP").Msgf("starting %d worker instances of service %v...", config.ncpuLimit, config.servicePath)
	for i := 0; i < config.ncpuLimit; i++ {
		log.Debug().Str("component", "IMP").Msgf("creating local socket and starting worker for thread %d...", i)
		socket := createSocket(i)
		connectionIndex := i / config.connShareFactor // 1 connection tuple will be shared by connShareFactor threads
		if i%config.connShareFactor == 0 {
			log.Debug().Str("component", "IMP").Msgf("creating task and result queue connection instance #%d...", connectionIndex)
			connections = append(connections, createQueueConnectionsAndTubes(config.queueConnectionString, config.taskQueueName, config.resultQueueName))
		}
		go startBatchConsumer(connections[connectionIndex], socket, errorCounter)
		go startSubprocess(config.servicePath, socket, subprocessCounter, errorCounter) // TODO: handle sub-process restarts on errors (use the go channel!)
	}

	forever := make(chan bool)
	<-forever // never end, inifitenly wait for all subprocesses
}
