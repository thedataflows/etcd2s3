package main

import (
	"os"

	"github.com/thedataflows/etcd2s3/cmd"
	log "github.com/thedataflows/go-lib-log"
)

var version = "dev"

func main() {
	err := cmd.Run(version, os.Args[1:])
	if err != nil {
		log.Errorf("main", err, "Command failed")
		os.Exit(1)
	}
}
