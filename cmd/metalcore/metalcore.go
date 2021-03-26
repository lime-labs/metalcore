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
	"go.uber.org/automaxprocs/maxprocs" // currently needed since cgroup CPU limits are not mapped to GOMAXPROCS automatically without it
)

var config struct {
	ncpuLimit             int
	errorLimit            int
	queueConnectionString string
	taskQueueName         string
	resultQueueName       string
	servicePath           string
}

// remove once protobuf is in place
type message struct {
	MessageID     string
	SessionID     string
	CorrelationID string
	Queue         string
	ReplyTo       string
	Payload       []byte
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

func handleRequest(socketConnection net.Conn, task []byte, results beanstalk.Tube) bool {
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

func getSocketPath(socketName string, number int) string {
	osTempDir := os.TempDir()
	name := socketName + strconv.Itoa(number) + ".sock"
	return filepath.Join(osTempDir, name)
}

func createSocketListener(queueConnection beanstalk.Conn, tasks beanstalk.TubeSet, results beanstalk.Tube, socket string, errorCounter chan<- int) {
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

			for {
				id, task, err := tasks.Reserve(1 * time.Hour) // get task from queue
				if err != nil {
					if err == beanstalk.ErrDeadline { // TODO/FIXME: this does NOT correctly check for a deadline error!
						log.Warn().Str("component", "IMP").Msgf("WARNING! Thread using socket %v tried to reserve a task from the queue but received DEADLINE SOON error, please handle it!", socket)
						// TODO: handle deadline soon errors properly!
					} else {
						log.Error().Err(err).Str("component", "IMP").Msg("error getting task from task queue")
					}
					continue // skip rest of the loop, begin next cycle
				}
				log.Trace().Str("component", "IMP").Msgf("received task #: %d, task payload: %d bytes from task queue(s)", id, len(task))

				if handleRequest(conn, task, results) { // if task was successfully handled and result properly submitted
					err := queueConnection.Delete(id)
					common.LogOnError(err, "error deleting task from task queue", "IMP")
					log.Trace().Str("component", "IMP").Msgf("succesfully deleted task #: %v", id)
				} else { // TODO: put task in failed queue/tube
					errorCounter <- 1
				}
			}
			//log.Fatal().Str("component", "IMP").Msg("message queue consumer closed - critical error, exiting now...")
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

	config.errorLimit = 10 // default value that can be overwritten with the env variable below
	if errorLimitString := os.Getenv("ERRORLIMIT"); errorLimitString != "" {
		var err error
		config.errorLimit, err = strconv.Atoi(errorLimitString)
		common.FailOnError(err, "error parsing ERRORLIMIT into an int", "IMP")
		log.Info().Str("component", "IMP").Msgf("error limit has been manually set to %d via env variable ERRORLIMIT", config.errorLimit)
	}
	log.Debug().Str("component", "IMP").Msgf("error limit is set to %d", config.errorLimit)

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
	taskConnection, err := beanstalk.Dial("tcp", config.queueConnectionString)
	common.FailOnError(err, "failed to connect to work queue server", "IMP")

	resultConnection, err := beanstalk.Dial("tcp", config.queueConnectionString)
	common.FailOnError(err, "failed to connect to work queue server", "IMP")

	taskTubeSet := beanstalk.NewTubeSet(taskConnection, config.taskQueueName)
	resultTube := beanstalk.NewTube(resultConnection, config.resultQueueName)

	subprocessCounter := make(chan int)   // go channel to count started subprocesses
	go subprocessCount(subprocessCounter) // start the goroutine that syncs the count
	errorCounter := make(chan int)        // go channel to count errors happening during sub-process execution
	go errorCount(errorCounter)           // start the goroutine that syncs the error count

	// create a go routine for each concurrent thread requested
	log.Info().Str("component", "IMP").Msgf("starting %d worker instances of service %v...", config.ncpuLimit, config.servicePath)
	for i := 0; i < config.ncpuLimit; i++ {
		log.Debug().Str("component", "IMP").Msgf("creating local socket and starting worker for thread %d...", i)
		socket := getSocketPath("metalcoreTaskSocket", i)
		createSocketListener(*taskConnection, *taskTubeSet, *resultTube, socket, errorCounter)
		// start actual worker binary process here, pass and read parameters/result via socket created above
		go startSubprocess(config.servicePath, socket, subprocessCounter, errorCounter) // TODO: handle sub-process restarts on errors (use the go channel!)
	}

	forever := make(chan bool)
	<-forever // never end, inifitenly wait for all subprocesses
}
