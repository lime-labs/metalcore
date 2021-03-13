package main

import (
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/lime-labs/metalcore/internal/pkg/common"
	"github.com/lime-labs/metalcore/pkg/queue"
	"github.com/streadway/amqp"
	_ "go.uber.org/automaxprocs" // currently needed since cgroup CPU limits are not mapped to GOMAXPROCS automatically without it
)

var config struct {
	debug                bool
	ncpulimit            int
	amqpConnectionString string
	taskQueueName        string
	prefetchCount        int
	resultBufferSize     int
	servicePath          string
}

func startSubprocess(pathToBinary string, socket string) bool {
	log.Println("[IMP]      starting process " + pathToBinary + " with socket " + socket)

	cmd := exec.Command(pathToBinary)

	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "METALCORESOCKET="+socket)

	// pass stdout and stderr to parent process
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	common.FailOnError(err, "Error starting subprocess")
	// this part will only be reached if the sub-process exits, .Run() has implicit wait() for process completion.
	// handle process restarts here or in parent function based on return code.
	return true
}

func handleRequest(socketConnection net.Conn, sessionID string, messageID string, task []byte, resultsBuffer chan<- queue.Message, resultQueue string, resultChannel *amqp.Channel) bool {
	if config.debug {
		log.Printf("[IMP]      [DEBUG] task messageID: %v | payload: %v", messageID, string(task))
	}
	// buffer to hold incoming data.
	buf := make([]byte, 1024)

	// Send a task back to the process that established the connection
	socketConnection.Write(task)

	len, err := socketConnection.Read(buf)
	common.FailOnError(err, "[IMP]      Error reading from socket")

	resultPayload := buf[:len]

	if config.debug {
		log.Printf("[IMP]      [DEBUG] result correlationID: %v | payload: %v", messageID, string(resultPayload))
	}

	if string(resultPayload) != "ERROR" {
		// write result to result buffer go channel
		result := queue.Message{CorrelationID: messageID, SessionID: sessionID, Queue: resultQueue, Payload: resultPayload}
		resultsBuffer <- result

		// signal successful handover to the result queue.
		// WARNING!: at this stage it's handed over to the result BUFFER, not yet submitted to the actual queue
		// on the broker side! might want to reconsider doing the confirm for the ACK more async as well.
		return true
	}

	log.Println("error during subprocess task execution", err)
	return false
}

func getSocketPath(socketName string, number int) string {
	osTempDir := os.TempDir()
	name := socketName + strconv.Itoa(number) + ".sock"
	return filepath.Join(osTempDir, name)
}

func createSocketListener(messages <-chan amqp.Delivery, resultsBuffer chan<- queue.Message, socket string, resultChannel *amqp.Channel) {
	log.Printf("[IMP]      creating UNIX domain socket at %s", socket)

	err := os.RemoveAll(socket)
	common.FailOnError(err, "failed to remove existing sockets")

	l, err := net.Listen("unix", socket)
	common.FailOnError(err, "socket listen error")

	go func() {
		defer l.Close()

		for {
			conn, err := l.Accept()
			common.FailOnError(err, "accept error")

			log.Printf("[IMP]      client process connected via [%s] on socket %s", conn.RemoteAddr().Network(), socket)

			for msg := range messages {
				// if the handler returns true then ACK, else NACK
				// to submit the message back into the rabbit queue for another round of processing
				if handleRequest(conn, msg.AppId, msg.MessageId, msg.Body, resultsBuffer, msg.ReplyTo, resultChannel) {
					msg.Ack(false) // true = ack all previously unacknowledged messages (batching of acks, go routine / thread fuckups ahead!),
				} else {
					msg.Nack(false, true)
				}
			}
			log.Fatalf("Rabbit consumer closed - critical Error")

		}
	}()
}

func init() {
	if os.Getenv("DEBUG") == "on" {
		config.debug = true
		log.Println("[IMP]      [DEBUG] mode enabled!")
	}

	ncpu := runtime.NumCPU()
	log.Printf("[IMP]      starting on a host with %d logical CPUs", ncpu)

	manualMaxCPULimit := os.Getenv("CPULIMIT")
	if manualMaxCPULimit != "" {
		var err error
		config.ncpulimit, err = strconv.Atoi(manualMaxCPULimit)
		common.FailOnError(err, "error parsing CPULIMIT into an int")
		runtime.GOMAXPROCS(config.ncpulimit)
		log.Printf("[IMP]      GOMAXPROCS has been manually set to %d via env variable CPULIMIT", config.ncpulimit)
	} else {
		config.ncpulimit = runtime.GOMAXPROCS(0) // leave limit untouched, return current setting of GOMAXPROCS
	}

	log.Printf("[IMP]      number of usable logical CPUs for this IMP is set to %d", config.ncpulimit)

	config.prefetchCount = config.ncpulimit * 100
	config.resultBufferSize = config.prefetchCount

	log.Printf("[IMP]      prefetch count for task queue set to %d messages", config.prefetchCount)
	log.Printf("[IMP]      buffer size for results in internal memory set to %d items", config.resultBufferSize)

	amqpHost := os.Getenv("AMQPHOST")
	amqpPort := os.Getenv("AMQPPORT")
	amqpUser := os.Getenv("AMQPUSER")
	amqpPassword := os.Getenv("AMQPPASSWORD")
	config.taskQueueName = os.Getenv("TASKQUEUENAME")
	config.servicePath = os.Getenv("SERVICEPATH")

	if amqpHost == "" || amqpUser == "" || amqpPassword == "" || config.taskQueueName == "" || config.servicePath == "" {
		log.Fatalln("[IMP]      not all required env variables for an AMQP connection set, exiting now...")
	}

	config.amqpConnectionString = "amqp://" + amqpUser + ":" + amqpPassword + "@" + amqpHost + ":" + amqpPort
}

func main() {
	taskChannel := queue.CreateConnectionChannel(config.amqpConnectionString, config.prefetchCount)
	taskQueue := queue.DeclareQueue(taskChannel, config.taskQueueName)

	resultChannel := queue.CreateConnectionChannel(config.amqpConnectionString, 0)

	msgs := queue.ConsumeOnChannel(taskChannel, taskQueue.Name)
	// create a buffer queue / go channel to handle all result submissions in a dedicated, central go routine
	resultsBuffer := make(chan queue.Message, config.prefetchCount) // could also be unbuffered since publishings are async by default, but this decouples it even more
	go queue.StartBackgroundPublisher(resultsBuffer, resultChannel)

	// create a go routine for each concurrent thread requested
	log.Printf("[IMP]      starting %d worker instances of service %v...", config.ncpulimit, config.servicePath)
	for i := 0; i < config.ncpulimit; i++ {
		log.Printf("[IMP]      creating local socket and starting worker for thread %d...", i)
		socket := getSocketPath("metalcoreTaskSocket", i)
		createSocketListener(msgs, resultsBuffer, socket, resultChannel)
		// start actual worker binary process here, pass and read parameters/result via socket created above
		go startSubprocess(config.servicePath, socket)
	}

	forever := make(chan bool)
	<-forever // never end, inifitenly wait for all subprocesses
}
