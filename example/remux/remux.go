package main

import (
	"flag"
	"reflect"
	"unsafe"

	"github.com/xueqing/ffmpeg-demo/demuxer"
	"github.com/xueqing/ffmpeg-demo/logutil"
	"github.com/xueqing/ffmpeg-demo/muxer"

	"github.com/google/logger"
	"github.com/xueqing/goav/libavcodec"
	"github.com/xueqing/goav/libavformat"
	"github.com/xueqing/goav/libavutil"
)

// refer ffmpeg/doc/examples/remuxing.c
func main() {
	var (
		verbose = flag.Bool("verbose", true, "print info level logs to stdout")
		logPath = flag.String("log", "remux.log", "file path to save log")

		iURL = flag.String("iurl", "/home/kiki/github/ffmpeg-demo/resource/movie.flv", "input url")
		iFmt = flag.String("ifmt", "flv", "input format")
		oURL = flag.String("ourl", "remux.flv", "output url")
		oFmt = flag.String("ofmt", "flv", "output format")

		demux *demuxer.Demuxer
		mux   *muxer.Muxer

		iStreams, oStreams []*libavformat.AvStream
	)
	flag.Parse()
	logutil.Init(*verbose, false, *logPath)
	defer logutil.Close()
	logger.Info("begin remux!")

	libavutil.AvLogSetLevel(48)

	// open demuxer url and muxer url
	if demux = demuxer.New(); demux == nil {
		logger.Errorf("New demuxer error")
		return
	}
	if err := demux.Open(*iURL, *iFmt); err != nil {
		logger.Errorf("demuxer Open error(%v)", err)
		return
	}
	defer demux.Close()

	if mux = muxer.New(); mux == nil {
		logger.Errorf("New muxer error")
		return
	}
	if err := mux.Open(*oURL, *oFmt); err != nil {
		logger.Errorf("muxer Open error(%v)", err)
		return
	}
	defer mux.Close()

	// copy stream
	if iStreams, _ = demux.Streams(); len(iStreams) == 0 {
		logger.Errorf("demuxer has 0 streams")
		return
	}

	for _, st := range iStreams {
		if err := mux.AddStream(st); err != nil {
			logger.Errorf("muxer AddStream error(%v)", err)
			return
		}
	}

	if oStreams, _ = mux.Streams(); len(oStreams) == 0 {
		logger.Errorf("muxer has 0 streams")
		return
	}

	if err := mux.WriteHeader(nil); err != nil {
		logger.Errorf("muxer WriteHeader error(%v)", err)
		return
	}

	// copy packets
	pkt := libavcodec.AvPacketAlloc()
	for {
		// get packet from demuxer
		if ret := demux.ReadPacket(pkt); ret < 0 {
			if ret == libavutil.AvErrorEOF {
				break
			}
			logger.Errorf("demuxer ReadPacket error(%v)", libavutil.ErrorFromCode(ret))
			return
		}
		defer pkt.AvPacketUnref()

		// modify pkt attributes
		iSt := iStreams[pkt.StreamIndex()]
		oSt := oStreams[pkt.StreamIndex()]
		if iSt.CodecParameters().CodecType() == libavcodec.AvMediaType(libavformat.AvmediaTypeVideo) {
			logPacket(pkt)
		}
		pkt.SetPts(libavcodec.AVRescaleQRnd(pkt.Pts(), iSt.TimeBase(), oSt.TimeBase(),
			libavcodec.AvRoundNearInf|libavcodec.AvRoundPassMinmax))
		pkt.SetDts(libavcodec.AVRescaleQRnd(pkt.Dts(), iSt.TimeBase(), oSt.TimeBase(),
			libavcodec.AvRoundNearInf|libavcodec.AvRoundPassMinmax))
		pkt.SetDuration(libavcodec.AVRescaleQRnd(int64(pkt.Duration()), iSt.TimeBase(), oSt.TimeBase(),
			libavcodec.AvRoundNearInf|libavcodec.AvRoundPassMinmax))
		if iSt.CodecParameters().CodecType() == libavcodec.AvMediaType(libavformat.AvmediaTypeVideo) {
			logPacket(pkt)
		}

		// send pkt to muxer
		if ret := mux.WritePacket(pkt); ret < 0 {
			logger.Errorf("muxer WritePacket error(%v)", libavutil.ErrorFromCode(ret))
			return
		}
	}

	if ret := mux.WriteTrailer(); ret < 0 {
		logger.Errorf("muxer WriteTrailer error(%v)", libavutil.ErrorFromCode(ret))
		return
	}
}

func logPacket(pkt *libavcodec.AvPacket) {
	logger.Infoln("===========")
	sli := reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(pkt.Data())),
		Len:  pkt.Size(),
		Cap:  pkt.Size(),
	}
	logger.Infof("%v %v %v %v", pkt.StreamIndex(), pkt.Pts(), pkt.Dts(), pkt.Duration())
	length := pkt.Size()
	if length > 16 {
		length = 16
	}
	buf := *(*[]byte)(unsafe.Pointer(&sli))
	logger.Infof("%x", buf[:length])
	logger.Infoln("===========")
}
