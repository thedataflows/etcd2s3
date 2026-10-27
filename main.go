package main

import (
	"os"

	"github.com/rs/zerolog/log"
	"github.com/thedataflows/etcd2s3/cmd"
)

var version = "dev"

func main() {
	err := cmd.Run(version, os.Args[1:])
	if err != nil {
		log.Error().Str("pkg", "main").Err(err).Msg("Command failed")
		os.Exit(1)
	}
}
