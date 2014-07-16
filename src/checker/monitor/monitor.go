package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/op/go-logging"

	"checker"
)

var (
	etcdHost = ""
	logLevel = "INFO"
	stdout   = false
	syslog   = true
)

var logger = logging.MustGetLogger("check")

func startMonitoring(cname string, etcdHost string) error {
	cont, err := checker.ContainerByName(cname)
	if err != nil {
		return err
	}
	monitor, err := checker.GetMonitor(cont, etcdHost)
	if err != nil {
		return err
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		for sig := range c {
			logger.Info("Got %s signal. Try to stop gracefuly.", sig)
			monitor.StopMonitoring()
		}
	}()

	monitor.StartMonitoring()

	return nil
}

func main() {
	flag.StringVar(&etcdHost, "etcd-host", "", "Host where etcd is listenting")
	flag.StringVar(&logLevel, "loglevel", "INFO", "Logging level. Must be one of (DEBUG, INFO, NOTICE, WARNING, ERROR, CRITICAL)")
	flag.BoolVar(&stdout, "stdout", false, "Write logs to STDOUT. Default false")
	flag.BoolVar(&syslog, "syslog", true, "Write logs to SYSLOG. Default true")

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Containers monitoring daemon.")
		fmt.Fprintln(os.Stderr, "Started daemon which monitors container and updates status in etcd.")
		fmt.Fprintf(os.Stderr, "Usage of check: monitor [options] container\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	checker.InitLogger(logLevel, stdout, syslog)
	//err := _checkLinked()
	cname := flag.Arg(0)
	if cname == "" {
		flag.Usage()
		os.Exit(2)
	}

	if etcdHost == "" {
		logger.Info("Argument -etcdHost doesn't set. Will try COREOS_PRIVATE_IPV4 eviroment variable")
		if addr, ok := checker.GetEnvVariable("COREOS_PRIVATE_IPV4"); ok {
			etcdHost = addr
		} else {
			log.Fatal("You have to set -etcd-host argument or COREOS_PRIVATE_IPV4 enviroment variable")

		}
	}
	err := startMonitoring(cname, etcdHost)
	if err != nil {
		logger.Error("Monitoring stopped with error %s", err)
	} else {
		logger.Info("Monitoring stopped.")
	}
}
