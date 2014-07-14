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
	logLevel = "INFO"
	stdout   = false
	syslog   = true
)

var logger = logging.MustGetLogger("checker")

func checkByName(cname string) error {
	cont, err := checker.ContainerByName(cname)
	if err != nil {
		return err
	}
	err = cont.Check()
	if err != nil {
		return err
	}
	return nil
}

func main() {
	flag.StringVar(&logLevel, "loglevel", "INFO", "Logging level. Must be one of (DEBUG, INFO, NOTICE, WARNING, ERROR, CRITICAL)")
	flag.BoolVar(&stdout, "stdout", true, "Write logs to STDOUT. Default true")
	flag.BoolVar(&syslog, "syslog", false, "Write logs to SYSLOG. Default false")

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Check service running in container.")
		fmt.Fprintf(os.Stderr, "Usage of check: check [options] container\n")
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
	err := checkByName(cname)
	if err != nil {
		log.Fatal(err)
	}
}
