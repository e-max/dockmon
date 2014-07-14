package main

import (
	"flag"
	"fmt"
	"log"
	"os"

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
	handler, err := checker.GetHandler(cname, etcdHost)
	if err != nil {
		return err
	}
	handler.StartMonitoring()
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
		if addr, ok := checker.GetEnvVariable("COREOS_PRIVATE_IPV4"); ok {
			etcdHost = addr
		} else {
			log.Fatal("You have to set -etcd-host argument or COREOS_PRIVATE_IPV4 enviroment variable")

		}
	}
	err := startMonitoring(cname, etcdHost)
	logger.Error("Monitoring stopped %s", err)
}
