package main

import (
	"flag"
	"os"

	"cfgkit/internal/logging"
	"cfgkit/internal/server"
)

func main() {
	configDir := flag.String("config", "./config", "")
	workDir := flag.String("workdir", "./", "")

	port := flag.String("port", "8080", "")

	flag.Parse()

	logger := logging.New()

	srv := server.New(*configDir, *workDir, *port, logger)

	if err := srv.Start(); err != nil {
		logger.Error("", "err", err)
		os.Exit(1)
	}
}
