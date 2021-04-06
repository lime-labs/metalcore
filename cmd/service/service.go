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

	socketType := os.Getenv("METALCORESOCKETTYPE")
	socketAddr := os.Getenv("METALCORESOCKETADDR")
	if socketAddr == "" || socketType == "" {
		log.Fatal().Str("component", "service").Msg("not enough metalcore task socket details provided via env variables METALCORESOCKETTYPE and METALCORESOCKETADDR, exiting now...")
	}

	log.Debug().Str("component", "service").Msgf("connecting sub-process instance on METALCORESOCKETADDR %v to parent IMP", socketAddr)

	socketConnection, err := net.Dial(socketType, socketAddr)
	common.FailOnError(err, "error connecting to socket", "service")
	defer socketConnection.Close()

	for {
		data := common.Reader(socketConnection, "service") // GET DATA

		sleepTask := &api.SleepExample{}
		err := proto.Unmarshal(data, sleepTask) // deserialize sleep task from data
		common.LogOnError(err, "failed to deserialize sleep example task", "service")
		log.Trace().Str("component", "service").Msgf("received sleep task: sleep duration: %d, payload size: %d bytes", sleepTask.Sleepduration, len(sleepTask.Fakepayload))

		result := sleepms(sleepTask.Sleepduration)         // DO STUFF
		common.Writer(socketConnection, result, "service") // DONE
	}
}
