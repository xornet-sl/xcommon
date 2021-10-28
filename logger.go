package xcommon

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
)

// SetInitialLogger configures initial logging system.
// TODO: make it more configurable
func SetInitialLogger(logLevel string, logFile string, trace bool) error {
	parsedLogLevel, err := log.ParseLevel(logLevel)
	if err != nil {
		return fmt.Errorf("Unknown log level: %s (should be one of [panic, fatal, error, warn, info, debug, trace])", logLevel)
	}
	log.SetLevel(parsedLogLevel)
	if trace {
		log.SetReportCaller(true)
	}
	if len(logFile) > 0 && logFile != "-" {
		_logFile, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("Unable to open log file '%s': %w", logFile, err)
		}
		log.SetOutput(_logFile)
	}
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:          true,
		DisableLevelTruncation: true,
		PadLevelText:           true,
	})
	log.Trace("Logger set")
	return nil
}
