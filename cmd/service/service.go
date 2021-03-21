package main

import (
	"io"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/lime-labs/metalcore/internal/pkg/common"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func reader(r io.Reader) []byte {
	buf := make([]byte, 1024)
	n, err := r.Read(buf[:])
	common.FailOnError(err, "socket read error", "service")
	data := buf[0:n]
	log.Trace().Str("component", "service").Msgf("sub-process instance received data from socket: %v", string(data))
	return data
}

func writer(w io.Writer, data []byte) {
	_, err := w.Write(data)
	common.FailOnError(err, "socket write error", "service")
	log.Trace().Str("component", "service").Msgf("sub-process instance sent result data to socket: %v", string(data))
}

func sleepms(data []byte) []byte {
	durationInt, _ := strconv.Atoi(string(data)) // need to convert to string first, currently raw serialization from/to strings
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

	c, err := net.Dial("unix", socket)
	common.FailOnError(err, "error connecting to socket", "service")
	defer c.Close()

	for {
		data := reader(c)       // GET TASK
		result := sleepms(data) // DO STUFF
		writer(c, result)       // DONE
	}
}
