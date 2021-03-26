package main

import (
	"net"
	"os"
	"time"

	"github.com/lime-labs/metalcore/internal/pkg/common"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func sleepms(data []byte) []byte {
	// durationInt, err := strconv.Atoi(string(data)) // need to convert to string first, currently raw serialization from/to strings
	// common.LogOnErrorShort(err, "error converting task payload to int, will be sleeping for 0s as a result...", "service")
	durationInt := 0 // TODO: remove the hardcoded value once protobuf is in place to read the proper value from the payload!
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
		data := common.Reader(socketConnection, "service") // reader(c)       // GET TASK
		result := sleepms(data)                            // DO STUFF
		common.Writer(socketConnection, result, "service") // writer(c, result)                   // DONE
	}
}
