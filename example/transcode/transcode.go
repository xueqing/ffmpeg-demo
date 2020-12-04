package main

import (
	"flag"
	"fmt"
	"unsafe"

	"github.com/xueqing/ffmpeg-demo/encoder"

	"github.com/google/logger"
	"github.com/xueqing/ffmpeg-demo/decoder"
	"github.com/xueqing/ffmpeg-demo/demuxer"
	"github.com/xueqing/ffmpeg-demo/logutil"
	"github.com/xueqing/ffmpeg-demo/muxer"
	"github.com/xueqing/goav/libavcodec"
	"github.com/xueqing/goav/libavformat"
	"github.com/xueqing/goav/libavutil"
)

type streamCtx struct {
	dec *decoder.Decoder
	enc *encoder.Encoder
}

var (
	demux  *demuxer.Demuxer
	mux    *muxer.Muxer
	stCtxs map[int]*streamCtx
)

// refer ffmpeg/doc/examples/transcoding.c
func main() {
	var (
		verbose = flag.Bool("verbose", true, "print info level logs to stdout")
		logPath = flag.String("log", "transcode.log", "file path to save log")

		iURL = flag.String("iurl", "/home/kiki/github/ffmpeg-demo/resource/movie.flv", "input url")
		iFmt = flag.String("ifmt", "flv", "input format")
		oURL = flag.String("ourl", "transcode.flv", "output url")
		oFmt = flag.String("ofmt", "flv", "output format")
	)
	flag.Parse()
	logutil.Init(*verbose, false, *logPath)
	defer logutil.Close()
	logger.Info("begin transcode!")

	// libavutil.AvLogSetLevel(48)
	defer closeResource()

	if err := openInput(*iURL, *iFmt); err != nil {
		logger.Errorf("openInput: error(%v)", err)
		return
	}
	if err := openOutput(*oURL, *oFmt); err != nil {
		logger.Errorf("openOutput: error(%v)", err)
		return
	}
	readAllPackets()
}

func openInput(iURL, iFmt string) (err error) {
	if demux = demuxer.New(); demux == nil {
		err = fmt.Errorf("New demuxer error")
		return
	}
	if err = demux.Open(iURL, iFmt); err != nil {
		return
	}
	iStreams, _ := demux.Streams()
	if len(iStreams) == 0 {
		err = fmt.Errorf("demuxer has 0 streams")
		return
	}
	stCtxs = make(map[int]*streamCtx)
	for i, st := range iStreams {
		if st.CodecParameters().CodecType() == libavutil.AvmediaTypeVideo ||
			st.CodecParameters().CodecType() == libavutil.AvmediaTypeAudio {
			dec := decoder.New(demux.InFormatContext())
			if dec == nil {
				err = fmt.Errorf("New decoder error")
				return
			}
			if err = dec.Open(st); err != nil {
				return
			}
			stCtxs[i] = &streamCtx{
				dec: dec,
			}
		}
	}
	return
}

func openOutput(oURL, oFmt string) (err error) {
	if mux = muxer.New(); mux == nil {
		err = fmt.Errorf("New muxer error")
		return
	}
	if err = mux.Open(oURL, oFmt); err != nil {
		return
	}
	iStreams, _ := demux.Streams()
	for idx, st := range iStreams {
		codecType := st.CodecParameters().CodecType()
		if codecType == libavutil.AvmediaTypeVideo ||
			codecType == libavutil.AvmediaTypeAudio {
			outSt, err := mux.AddStream(st)
			if err != nil {
				return err
			}
			setStreamContext(st, outSt)
		} else if codecType == libavutil.AvmediaTypeUnknown {
			err = fmt.Errorf("openOutput: stream(%d) is of unknown type", idx)
			return
		}
	}
	if err = mux.WriteHeader(nil); err != nil {
		logger.Errorf("openOutput: muxer WriteHeader error(%v)", err)
		return
	}
	return
}

func closeResource() {
	if stCtxs != nil {
		for stIdx, stCtx := range stCtxs {
			if stCtx.dec != nil {
				logger.Infof("closeResource: close decoder of streamIndex(%v)", stIdx)
				stCtx.dec.Close()
			}
			if stCtx.enc != nil {
				logger.Infof("closeResource: close encoder of streamIndex(%v)", stIdx)
				stCtx.enc.Close()
			}
		}
	}
	if demux != nil {
		logger.Infof("closeResource: close demuxer")
		demux.Close()
	}
	if mux != nil {
		logger.Infof("closeResource: close muxer")
		mux.Close()
	}
}

func setStreamContext(pInStream, pOutStream *libavformat.AvStream) (err error) {
	enc := encoder.New()
	if enc == nil {
		err = fmt.Errorf("New encoder error")
		return
	}
	if err = enc.Open(pInStream); err != nil {
		return
	}

	pDecCtx := (*libavcodec.AvCodecContext)(unsafe.Pointer(pInStream.Codec()))
	pEncCtx := enc.EncCodecContext()
	pEnc := enc.EncCodec()

	// In this example, we transcode to same properties (picture size, sample rate etc.).
	// These properties can be changed for output streams easily using filters
	if pDecCtx.CodecType() == libavutil.AvmediaTypeVideo {
		pEncCtx.SetHeight(pDecCtx.Height())
		pEncCtx.SetWidth(pDecCtx.Width())
		pEncCtx.SetSampleAspectRatio(pDecCtx.SampleAspectRatio())
		// take first format from list of supported formats
		if pixFmts := pEnc.PixFmts(); pixFmts != nil {
			pEncCtx.SetPixelFormat(pixFmts[0])
		} else {
			pEncCtx.SetPixelFormat(pDecCtx.PixFmt())
		}
		// video time_base can be set to whatever is handy and supported by encoder
		pEncCtx.SetTimebase(libavcodec.AvInvQ(pDecCtx.Framerate()))
	} else {
		pEncCtx.SetSampleRate(pDecCtx.SampleRate())
		pEncCtx.SetChannelLayout(pDecCtx.ChannelLayout())
		pEncCtx.SetChannels(pDecCtx.Channels())
		// take first format from list of supported formats
		pEncCtx.SetSampleFmt(pEnc.SampleFmts()[0])
		pEncCtx.SetTimebase(libavcodec.NewAvRational(1, pEncCtx.SampleRate()))
	}

	if (mux.OutFormatContext().Flags() & libavformat.AvfmtGlobalheader) != 0 {
		pEncCtx.SetFlags(pEncCtx.Flags() | libavcodec.AvCodecFlagGlobalHeader)
	}

	// Third parameter can be used to pass settings to encoder
	if ret := pEncCtx.AvcodecOpen2(pEnc, nil); ret < 0 {
		err = fmt.Errorf("setStreamContext: open encoder error(%v)", libavutil.ErrorFromCode(ret))
		return
	}
	if ret := pEncCtx.AvcodecParametersFromContext(pOutStream.CodecParameters()); ret < 0 {
		err = fmt.Errorf("setStreamContext: copy encoder parameters to ouyput stream error(%v)", libavutil.ErrorFromCode(ret))
		return
	}
	pOutStream.SetTimeBase(pEncCtx.TimeBase())

	stCtxs[pInStream.Index()].enc = enc
	return
}

func readAllPackets() (err error) {
	var (
		gotFrame int
		pFrame   *libavutil.AvFrame
		pPkt     *libavcodec.AvPacket
	)

	pPkt = libavcodec.AvPacketAlloc()
	iStreams, _ := demux.Streams()
	for {
		if ret = demux.ReadPacket(pPkt); ret < 0 {
			err = fmt.Errorf("demuxer ReadPacket error(%v)", libavutil.ErrorFromCode(ret))
			break
		}
		defer pPkt.AvPacketUnref()

		stIdx := pPkt.StreamIndex()
		logger.Infof("demuxer read frame of streamIndex(%v)", stIdx)

		logger.Infof("decoding packet ...")
		pDecCtx := stCtxs[stIdx].dec.DecCodecContext()
		pPkt.AvPacketRescaleTs(iStreams[stIdx].TimeBase(), pDecCtx.TimeBase())
		if pFrame, gotFrame, err = stCtxs[stIdx].dec.DecodePacket(pPkt); err != nil {
			logger.Errorf("error(%v)", err)
			break
		}
		defer libavutil.AvFrameFree(pFrame)
		if gotFrame == 1 {
			if err = encoderWriteFrame(pFrame, stIdx, nil); err != nil {
				break
			}
		}
	}

	for stIdx := range iStreams {
		if err = flushEncoder(stIdx); err != nil {
			logger.Errorf("flush encoder of streamIndex(%v) error(%v)", stIdx, err)
			return
		}
	}

	mux.WriteTrailer()
	return
}

func flushEncoder(stIdx int) (err error) {
	var (
		gotFrame int
	)
	if (stCtxs[stIdx].enc.EncCodecContext().Codec().Capabilities() & libavcodec.AvCodecCapDelay) == 0 {
		return
	}

	for {
		logger.Infof("flushEncoder: streamIndex(%v)", stIdx)
		if err = encoderWriteFrame(nil, stIdx, &gotFrame); err != nil {
			break
		}
		if gotFrame == 0 {
			return
		}
	}
	return
}

func encoderWriteFrame(pFrame *libavutil.AvFrame, stIdx int, gotFrame *int) (err error) {
	var (
		pEncPkt *libavcodec.AvPacket
	)

	// encode frame
	logger.Infof("encoding frame")
	pFrameConvert := (*libavcodec.AvFrame)(unsafe.Pointer(pFrame))
	pEncPkt, err = stCtxs[stIdx].enc.EncodeFrame(pFrameConvert, gotFrame)
	if err != nil || pEncPkt == nil {
		return
	}
	defer pEncPkt.AvPacketUnref()

	// prepare packet for muxer
	pEncCtx := stCtxs[stIdx].enc.EncCodecContext()
	pEncPkt.SetStreamIndex(stIdx)
	pEncPkt.AvPacketRescaleTs(pEncCtx.TimeBase(), mux.OutFormatContext().Streams()[stIdx].TimeBase())

	// mux encoded frame
	logger.Infof("mux frame")
	if ret := mux.IntervedWritePacket(pEncPkt); ret < 0 {
		err = fmt.Errorf("encoderWriteFrame: IntervedWritePacket error(%v)", libavutil.ErrorFromCode(ret))
		return
	}

	return
}
