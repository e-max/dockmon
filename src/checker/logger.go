package checker

import (
	"fmt"
	"log"
	"os"

	"github.com/op/go-logging"
)

var logger = logging.MustGetLogger("checker")

var logLevels = map[string]logging.Level{
	"DEBUG":    logging.DEBUG,
	"INFO":     logging.INFO,
	"NOTICE":   logging.NOTICE,
	"WARNING":  logging.WARNING,
	"ERROR":    logging.ERROR,
	"CRITICAL": logging.CRITICAL,
}

func InitLogger(level string, stdout bool, syslog bool) {
	logging.SetFormatter(logging.MustStringFormatter("▶ %{level} %{module} %{message}"))
	backends := []logging.Backend{}

	if !(stdout || syslog) {
		fmt.Errorf("You have to set stdout or syslog options")
	}

	if syslog {
		syslogBackend, err := logging.NewSyslogBackend("checker")
		if err != nil {
			logger.Fatal(err)
		}
		backends = append(backends, syslogBackend)
	}

	if stdout {
		stderrBackend := logging.NewLogBackend(os.Stderr, "", log.LstdFlags|log.Lshortfile)
		stderrBackend.Color = true
		backends = append(backends, stderrBackend)
	}

	logging.SetBackend(backends...)
	lvl, ok := logLevels[level]
	if !ok {
		log.Fatal("Loglevel must be one of (DEBUG, INFO, NOTICE, WARNING, ERROR, CRITICAL)")
	}
	logging.SetLevel(lvl, "checker")
	logging.SetLevel(lvl, "checker.check")
	logging.SetLevel(lvl, "checker.monitor")
	logging.SetLevel(lvl, "checker.events")

}
