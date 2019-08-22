package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/j-forster/mqtt"
)

func main() {

	verbose := flag.Bool("v", false, "Verbose logging")
	debug := flag.Bool("d", false, "Debug logging")
	silent := flag.Bool("s", false, "Silent / No logging")
	warnings := flag.Bool("w", false, "Warnings logging")
	pprof := flag.Bool("pprof", false, "Enable pprof ('localhost:6060')")

	addr := flag.String("a", ":1883", "MQTT Listen Address")

	flag.Parse()

	var logger *log.Logger
	var logLevel mqtt.LogLevel

	if !*silent {
		logger = log.New(os.Stdout, "", 0)
		if *verbose {
			logLevel = mqtt.LogLevelVerbose
		} else if *warnings {
			logLevel = mqtt.LogLevelWarnings
		} else if *debug {
			logLevel = mqtt.LogLevelDebug
		}
	}

	if *pprof {
		go func() {
			runtime.SetBlockProfileRate(1)
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	// server := mqtt.NewServer(nil, nil)
	// mqtt.ListenAndServe(":1883", server)

	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, syscall.SIGTERM, syscall.SIGINT)

	server := mqtt.NewServer(nil, logger, logLevel)

	go func() {
		<-shutdown
		server.Close()
		os.Exit(0)
	}()

	logger.Fatal(mqtt.ListenAndServe(*addr, server))
}
