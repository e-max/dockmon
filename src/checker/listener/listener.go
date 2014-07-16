package main

import (
	"checker"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/op/go-logging"
)

var (
	etcdHost = ""
	logLevel = "INFO"
	stdout   = false
	syslog   = true
)

var logger = logging.MustGetLogger("check")

func main() {
	flag.StringVar(&etcdHost, "etcd-host", "", "Host where etcd is listenting")
	flag.StringVar(&logLevel, "loglevel", "INFO", "Logging level. Must be one of (DEBUG, INFO, NOTICE, WARNING, ERROR, CRITICAL)")
	flag.BoolVar(&stdout, "stdout", false, "Write logs to STDOUT. Default false")
	flag.BoolVar(&syslog, "syslog", true, "Write logs to SYSLOG. Default true")

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Docker event listener.")
		fmt.Fprintln(os.Stderr, "Started daemon which monitors container and updates status in etcd.")
		fmt.Fprintf(os.Stderr, "Usage of check: listener [options] container\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	checker.InitLogger(logLevel, stdout, syslog)

	if etcdHost == "" {
		logger.Info("Argument -etcdHost doesn't set. Will try COREOS_PRIVATE_IPV4 eviroment variable")
		if addr, ok := checker.GetEnvVariable("COREOS_PRIVATE_IPV4"); ok {
			etcdHost = addr
		} else {
			log.Fatal("You have to set -etcd-host argument or COREOS_PRIVATE_IPV4 enviroment variable")

		}
	}
	listener, err := checker.GetListener(etcdHost)
	if err != nil {
		logger.Error("Cannot start listener : %s", err)
		os.Exit(2)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		for sig := range c {
			logger.Info("Got %s signal. Try to stop gracefuly.", sig)
			listener.Stop()
		}
	}()

	err = listener.Run()
	if err != nil {
		logger.Error("Monitoring stopped with error %s", err)
	} else {
		logger.Info("Monitoring stopped ")
	}

}
