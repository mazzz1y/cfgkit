package main

import (
	"flag"
	"os"
	"runtime"

	"cfgkit/internal/logging"
	"cfgkit/internal/server"
)

func main() {
	configDir := flag.String("config", "./config", "")
	workDir := flag.String("workdir", "./", "")

	port := flag.String("port", "8080", "")
	maxChecks := flag.Int("max-checks", runtime.NumCPU(), "maximum number of concurrent check commands")

	flag.Parse()

	logger := logging.New()

	srv := server.New(*configDir, *workDir, *port, *maxChecks, logger)

	if err := srv.Start(); err != nil {
		logger.Error("", "err", err)
		os.Exit(1)
	}
}
