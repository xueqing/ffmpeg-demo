package main

import (
	"flag"
	"os"

	"github.com/giorgisio/goav/avformat"
	"github.com/google/logger"
)

const logPath = "ffmpeg-demo.log"

var verbose = flag.Bool("verbose", false, "print info level logs to stdout")

func main() {
	flag.Parse()

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		logger.Fatalf("failed to open log file error(%v)", err)
	}
	defer file.Close()

	defer logger.Init("FFmpegDemoLogger", *verbose, true, file).Close()

	logger.Info("begin main!")
	goavTest()
}

func goavTest() {
	filename := "resource/movie.flv"

	// register all formats and codecs
	avformat.AvRegisterAll()

	ctx := avformat.AvformatAllocContext()

	// open video file
	if avformat.AvformatOpenInput(&ctx, filename, nil, nil) != 0 {
		logger.Warningf("failed to open file(%v)", filename)
		return
	}

	// remember to close input file and free context
	defer ctx.AvformatCloseInput()

	// retrive stream info
	if ctx.AvformatFindStreamInfo(nil) < 0 {
		logger.Warningf("failed to find stream info")
		return
	}

	logger.Infof("stream(%v)", ctx.NbStreams())
}
