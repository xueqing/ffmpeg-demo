package main

import (
	"flag"
	"reflect"
	"unsafe"

	"github.com/xueqing/ffmpeg-demo/demuxer"
	"github.com/xueqing/ffmpeg-demo/logutil"
	"github.com/xueqing/ffmpeg-demo/muxer"

	"github.com/giorgisio/goav/avcodec"
	"github.com/giorgisio/goav/avformat"
	"github.com/giorgisio/goav/avutil"
	"github.com/google/logger"
)

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

		iStreams, oStreams []*avformat.Stream
	)
	flag.Parse()
	logutil.Init(*verbose, false, *logPath)
	defer logutil.Close()
	logger.Info("begin remux!")

	avutil.AvLogSetLevel(4)

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
	pkt := avcodec.AvPacketAlloc()
	for {
		// get packet from demuxer
		if ret := demux.ReadPacket(pkt); ret < 0 {
			if ret == avutil.AvErrorEOF {
				break
			}
			logger.Errorf("demuxer ReadPacket error(%v)", avutil.ErrorFromCode(ret))
			return
		}
		defer pkt.AvPacketUnref()

		// modify pkt attributes
		iSt := iStreams[pkt.StreamIndex()]
		oSt := oStreams[pkt.StreamIndex()]
		if iSt.CodecParameters().AvCodecGetType() == avcodec.MediaType(avformat.AVMEDIA_TYPE_VIDEO) {
			logPacket(pkt)
		}
		pkt.SetPts(avcodec.AVRescaleQRnd(pkt.Pts(), iSt.TimeBase(), oSt.TimeBase(), avcodec.AV_ROUND_NEAR_INF|avcodec.AV_ROUND_PASS_MINMAX))
		pkt.SetDts(avcodec.AVRescaleQRnd(pkt.Dts(), iSt.TimeBase(), oSt.TimeBase(), avcodec.AV_ROUND_NEAR_INF|avcodec.AV_ROUND_PASS_MINMAX))
		pkt.SetDuration(avcodec.AVRescaleQRnd(int64(pkt.Duration()), iSt.TimeBase(), oSt.TimeBase(), avcodec.AV_ROUND_NEAR_INF|avcodec.AV_ROUND_PASS_MINMAX))
		if iSt.CodecParameters().AvCodecGetType() == avcodec.MediaType(avformat.AVMEDIA_TYPE_VIDEO) {
			logPacket(pkt)
		}

		// send pkt to muxer
		if ret := mux.WritePacket(pkt); ret < 0 {
			logger.Errorf("muxer WritePacket error(%v)", avutil.ErrorFromCode(ret))
			return
		}
	}

	if ret := mux.WriteTrailer(); ret < 0 {
		logger.Errorf("muxer WriteTrailer error(%v)", avutil.ErrorFromCode(ret))
		return
	}
}

func logPacket(pkt *avcodec.Packet) {
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
