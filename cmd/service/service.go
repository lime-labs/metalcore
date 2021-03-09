package main

import (
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/lime-labs/metalcore/internal/pkg/common"
)

func reader(r io.Reader) []byte {
	buf := make([]byte, 1024)
	n, err := r.Read(buf[:])
	common.FailOnError(err, "[instance] socket read error")
	data := buf[0:n]
	//log.Println("[instance] subprocess received data:", string(data))
	return data
}

func writer(w io.Writer, data []byte) {
	_, err := w.Write([]byte(data))
	common.FailOnError(err, "[instance] socket write error")
}

func sleepms(data []byte) string {
	// need to convert to string first, currently raw serialization from/to strings
	durationInt, _ := strconv.Atoi(string(data))
	duration := time.Millisecond * time.Duration(durationInt)

	time.Sleep(duration)

	return "worker process slept for " + duration.String()
}

func main() {

	socket := os.Getenv("METALCORESOCKET")

	if socket == "" {
		log.Fatalln("[instance] no metalcore task socket provided via env variable METALCORESOCKET, exiting now...")
	}

	log.Println("[instance] connecting sub process instance on METALCORESOCKET:", socket)

	c, err := net.Dial("unix", socket)
	common.FailOnError(err, "[instance] error connecting to socket")
	defer c.Close()

	for {
		// GET TASK
		data := reader(c)

		// DO STUFF
		result := sleepms(data)

		// DONE
		writer(c, []byte(result))
	}
}
