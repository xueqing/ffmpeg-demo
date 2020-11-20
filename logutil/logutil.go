package logutil

import (
	"os"

	"github.com/google/logger"
)

var pLogger *logger.Logger

// Init init logger
func Init(verbose, systemLog bool, logPath string) {
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		logger.Fatalf("logger Init: open log file(%v) error(%v)", logPath, err)
	}
	pLogger = logger.Init("FFmpegDemoLogger", verbose, systemLog, file)
}

// Close close logger
func Close() {
	pLogger.Close()
}
