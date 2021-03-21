package common

import (
	"github.com/rs/zerolog/log"
)

// FailOnError provides a shortcut for checking for errors and failing fataly in one clean line
func FailOnError(err error, msg string, comp string) {
	if err != nil {
		log.Fatal().Err(err).Str("component", comp).Msg(msg)
	}
}
