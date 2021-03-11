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
)

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

func handleRequest(socketConnection net.Conn, sessionID string, messageID string, task []byte, resultsBuffer chan<- queue.Result, resultQueue string, resultChannel *amqp.Channel) bool {
	// buffer to hold incoming data.
	buf := make([]byte, 1024)

	// Send a task back to the process that established the connection
	socketConnection.Write(task)

	len, err := socketConnection.Read(buf)
	common.FailOnError(err, "[IMP]      Error reading from socket")

	resultPayload := buf[:len]

	if string(resultPayload) != "ERROR" {
		// write result to result buffer go channel
		result := queue.Result{MessageID: messageID, SessionID: sessionID, ResultQueue: resultQueue, Payload: resultPayload}
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

func createSocketListener(messages <-chan amqp.Delivery, resultsBuffer chan queue.Result, socket string, resultChannel *amqp.Channel) {
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

func main() {
	ncpu := runtime.NumCPU()
	log.Printf("[IMP]      starting on a host / container with %d threads", ncpu)

	amqpHost := os.Getenv("AMQPHOST")
	amqpPort := os.Getenv("AMQPPORT")
	amqpUser := os.Getenv("AMQPUSER")
	amqpPassword := os.Getenv("AMQPPASSWORD")
	taskQueueName := os.Getenv("TASKQUEUENAME")
	servicePath := os.Getenv("SERVICEPATH")

	if amqpHost == "" || amqpUser == "" || amqpPassword == "" || taskQueueName == "" || servicePath == "" {
		log.Fatalln("[IMP]      not all required env variables set, exiting now...")
	}

	amqpConnectionString := "amqp://" + amqpUser + ":" + amqpPassword + "@" + amqpHost + ":" + amqpPort

	prefetchCount := ncpu * 100
	taskChannel := queue.CreateConnectionChannel(amqpConnectionString, prefetchCount)
	taskQueue := queue.DeclareQueue(taskChannel, taskQueueName)

	resultChannel := queue.CreateConnectionChannel(amqpConnectionString, 0)

	msgs := queue.ConsumeOnChannel(taskChannel, taskQueue.Name)
	// create a buffer queue / go channel to handle all result submissions in one thread / go routine
	resultsBuffer := make(chan queue.Result, prefetchCount) // could also be unbuffered since publishings are async by default, but this decouples it even more
	go queue.StartBackgroundResultsPublisher(resultsBuffer, resultChannel)

	// create a goroutine for the number of concurrent threads requested
	for i := 0; i < ncpu; i++ {
		log.Printf("[IMP]      creating socket and starting worker for thread %v...\n", i)
		socket := getSocketPath("metalcoreTaskSocket", i)
		createSocketListener(msgs, resultsBuffer, socket, resultChannel)
		// start actual worker binary process here, pass and read parameters/result via socket created above
		go startSubprocess(servicePath, socket)
	}

	// never end, inifitenly wait for all subprocesses
	forever := make(chan bool)
	<-forever
}
