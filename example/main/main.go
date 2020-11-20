package main

import (
	"flag"

	"github.com/xueqing/ffmpeg-demo/logutil"

	"github.com/giorgisio/goav/avformat"
	"github.com/google/logger"
)

func main() {
	var (
		verbose = flag.Bool("verbose", false, "print info level logs to stdout")
		logPath = flag.String("log", "ffmpeg-demo.log", "file path to save log")
	)
	flag.Parse()

	logutil.Init(*verbose, false, *logPath)
	defer logutil.Close()

	logger.Info("begin main!")
	goavTest()
}

func goavTest() {
	filename := "../../resource/movie.flv"

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
