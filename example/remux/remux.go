package main

import (
	"flag"
	"fmt"
	"io"
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

	// libavutil.AvLogSetLevel(48)

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
		outSt, err := mux.AddStream(st)
		if err != nil {
			logger.Errorf("Muxer AddStream error(%v)", err)
			return
		}
		if err := setStreamContext(st, outSt); err != nil {
			logger.Errorf("setStreamContext error(%v)", err)
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
		if err := demux.ReadPacket(pkt); err != nil {
			if err == io.EOF {
				break
			}
			logger.Errorf("demuxer ReadPacket error(%v)", err)
			return
		}
		defer pkt.AvPacketUnref()

		// modify pkt attributes
		iSt := iStreams[pkt.StreamIndex()]
		oSt := oStreams[pkt.StreamIndex()]
		if iSt.CodecParameters().CodecType() == libavcodec.AvMediaType(libavutil.AvmediaTypeVideo) {
			// logPacket(pkt)
		}
		pkt.SetPts(libavcodec.AVRescaleQRnd(pkt.Pts(), iSt.TimeBase(), oSt.TimeBase(),
			libavcodec.AvRoundNearInf|libavcodec.AvRoundPassMinmax))
		pkt.SetDts(libavcodec.AVRescaleQRnd(pkt.Dts(), iSt.TimeBase(), oSt.TimeBase(),
			libavcodec.AvRoundNearInf|libavcodec.AvRoundPassMinmax))
		pkt.SetDuration(libavcodec.AVRescaleQRnd(int64(pkt.Duration()), iSt.TimeBase(), oSt.TimeBase(),
			libavcodec.AvRoundNearInf|libavcodec.AvRoundPassMinmax))
		if iSt.CodecParameters().CodecType() == libavcodec.AvMediaType(libavutil.AvmediaTypeVideo) {
			// logPacket(pkt)
		}

		// send pkt to muxer
		if err := mux.WritePacket(pkt); err != nil {
			logger.Errorf("muxer WritePacket error(%v)", err)
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

func setStreamContext(pInStream, pOutStream *libavformat.AvStream) (err error) {
	var (
		pEnc    *libavcodec.AvCodec
		pEncCtx *libavcodec.AvCodecContext
	)

	pDecCtx := (*libavcodec.AvCodecContext)(unsafe.Pointer(pInStream.Codec()))
	codecID := libavcodec.AvCodecID(pDecCtx.CodecID())
	// Find a registered encoder with a matching codec ID.
	if pEnc = libavcodec.AvcodecFindEncoder(codecID); pEnc == nil {
		err = fmt.Errorf("setStreamContext: find encoder by id(%v) error", libavcodec.AvcodecGetName(codecID))
		return
	}

	// Allocate an AVCodecContext and set its fields to default values. The
	// resulting struct should be freed with avcodec_free_context().
	if pEncCtx = pEnc.AvcodecAllocContext3(); pEncCtx == nil {
		err = fmt.Errorf("setStreamContext: alloc encoder context error")
		return
	}
	defer pEncCtx.AvcodecFreeContext()

	// copy parameters from context
	if ret := pEncCtx.AvcodecParametersFromContext(pOutStream.CodecParameters()); ret < 0 {
		err = fmt.Errorf("setStreamContext: copy encoder parameters to ouyput stream error(%v)", libavutil.ErrorFromCode(ret))
		return
	}

	pOutStream.SetPrivData(pInStream.PrivData())
	pOutStream.SetTimeBase(pInStream.TimeBase())
	pOutStream.SetStartTime(pInStream.StartTime())
	pOutStream.SetDuration(pInStream.Duration())
	pOutStream.SetNbFrames(pInStream.NbFrames())
	pOutStream.SetDisposition(pInStream.Disposition())
	pOutStream.SetDiscard(pInStream.Discard())
	pOutStream.SetRFrameRate(pInStream.RFrameRate())
	pOutStream.CodecParameters().AvcodecParametersCopy(pInStream.CodecParameters())

	switch codecType := pInStream.CodecParameters().CodecType(); codecType {
	case libavutil.AvmediaTypeVideo:
		pOutStream.CodecParameters().SetHeight(pInStream.CodecParameters().Height())
		pOutStream.CodecParameters().SetWidth(pInStream.CodecParameters().Width())
	case libavutil.AvmediaTypeAudio:
		pOutStream.CodecParameters().SetSampleRate(pInStream.CodecParameters().SampleRate())
		pOutStream.CodecParameters().SetChannels(pInStream.CodecParameters().Channels())
		pOutStream.CodecParameters().SetChannelLayout(pInStream.CodecParameters().ChannelLayout())
		pOutStream.CodecParameters().SetFormat(pInStream.CodecParameters().Format())
	default:
		codecTypeConvert := libavutil.AvMediaType(codecType)
		logger.Warningf("setStreamContext: unsupported media type(%v)", libavutil.AvGetMediaTypeString(codecTypeConvert))
	}
	return
}
