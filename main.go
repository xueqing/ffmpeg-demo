package main

import (
	"flag"
	"os"

	"github.com/google/logger"
)

const logPath = "ffmpeg-demo.log"

var verbose = flag.Bool("verbose", false, "print info level logs to stdout")

func main() {
	flag.Parse()

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		logger.Fatalf("Failed to open log file error(%v)", err)
	}
	defer file.Close()

	defer logger.Init("FFmpegDemoLogger", *verbose, true, file).Close()

	logger.Info("begin main!")
}
