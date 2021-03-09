package common

import (
	"log"
)

// FailOnError provides a shortcut for checking for errors and failing fataly in one clean line
func FailOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
