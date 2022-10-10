package xcommon

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	// prefixed "github.com/x-cray/logrus-prefixed-formatter"
	prefixed "github.com/hu13/logrus-prefixed-formatter" // log-file-line branch
)

// SetInitialLogger configures initial logging system.
func SetInitialLogger(logLevel string, logFile string, trace bool) error {
	parsedLogLevel, err := log.ParseLevel(logLevel)
	if err != nil {
		return fmt.Errorf("Unknown log level: %s (should be one of [panic, fatal, error, warn, info, debug, trace])", logLevel)
	}
	log.SetLevel(parsedLogLevel)
	log.SetReportCaller(trace)
	if len(logFile) > 0 && logFile != "-" {
		_logFile, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("Unable to open log file '%s': '%w'", logFile, err)
		}
		log.SetOutput(_logFile)
	}
	log.SetFormatter(&prefixed.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05.000",
		FullTimestamp:   true,
		ForceFormatting: true,
	})
	log.Trace("Logger set")
	return nil
}
