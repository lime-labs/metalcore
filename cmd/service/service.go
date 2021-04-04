package main

import (
	"net"
	"os"
	"time"

	api "github.com/lime-labs/metalcore/api/v1"
	"github.com/lime-labs/metalcore/internal/pkg/common"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

func deserializeTask(data []byte) (task *api.Task, err error) {
	task = &api.Task{}
	err = proto.Unmarshal(data, task)
	return
}

func sleepms(durationInt int32) []byte {
	duration := time.Millisecond * time.Duration(durationInt)
	time.Sleep(duration) // the actual task at hand
	result := "worker process slept for " + duration.String()
	log.Trace().Str("component", "service").Msgf("sub-process instance calculated result: %v", result)
	return []byte(result)
}

func main() {
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

	socket := os.Getenv("METALCORESOCKET")
	if socket == "" {
		log.Fatal().Str("component", "service").Msg("no metalcore task socket provided via env variable METALCORESOCKET, exiting now...")
	}

	log.Debug().Str("component", "service").Msgf("connecting sub-process instance on METALCORESOCKET %v to parent IMP", socket)

	socketConnection, err := net.Dial("unix", socket)
	common.FailOnError(err, "error connecting to socket", "service")
	defer socketConnection.Close()

	for {
		data := common.Reader(socketConnection, "service") // GET DATA

		task, err := deserializeTask(data) // DESERIALIZE DATA
		common.LogOnError(err, "failed to deserialize task", "service")
		log.Trace().Str("component", "service").Msgf("received task: %d, session: %v, sleep duration: %d, payload size: %d bytes", task.Id, task.Session, task.Sleepduration, len(task.Payload))

		result := sleepms(task.Sleepduration)              // DO STUFF
		common.Writer(socketConnection, result, "service") // DONE
	}
}
