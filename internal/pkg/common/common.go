package common

import (
	"encoding/binary"
	"io"

	"github.com/rs/zerolog/log"
)

// LogOnError provides a shortcut for checking for errors and logging them in one clean line, including the err content itself
func LogOnError(err error, msg string, comp string) {
	if err != nil {
		log.Error().Err(err).Str("component", comp).Msg(msg)
	}
}

// LogOnErrorShort provides a shortcut for checking for errors and logging them in one clean line, without the err content included
func LogOnErrorShort(err error, msg string, comp string) {
	if err != nil {
		log.Error().Str("component", comp).Msg(msg)
	}
}

// FailOnError provides a shortcut for checking for errors and failing fataly in one clean line
func FailOnError(err error, msg string, comp string) {
	if err != nil {
		log.Fatal().Err(err).Str("component", comp).Msg(msg)
	}
}

// Reader...
func Reader(r io.Reader, comp string) []byte {
	var payloadSize uint64
	err := binary.Read(r, binary.LittleEndian, &payloadSize)
	LogOnError(err, "error reading header from socket", comp)
	log.Trace().Str("component", comp).Msgf("%v received header from socket, payload size is %d bytes", comp, payloadSize)

	dataBuffer := make([]byte, payloadSize) // target buffer for the data
	n, err := io.ReadFull(r, dataBuffer)    // fill the buffer with the data
	FailOnError(err, "socket read error", comp)
	log.Trace().Str("component", comp).Msgf("%v received %d bytes of data from socket", comp, n)
	return dataBuffer
}

// Writer...
func Writer(w io.Writer, data []byte, comp string) {
	payloadSize := uint64(len(data))
	err := binary.Write(w, binary.LittleEndian, &payloadSize)
	LogOnError(err, "error sending result header to socket", comp)
	log.Trace().Str("component", comp).Msgf("%v sent header via socket, payload size is %d bytes", comp, payloadSize)

	n, err := w.Write(data)
	FailOnError(err, "socket write error", comp)
	log.Trace().Str("component", comp).Msgf("%v sent %d bytes of data to socket", comp, n)
}
