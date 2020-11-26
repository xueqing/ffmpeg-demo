package muxer

import (
	"fmt"
	"unsafe"

	"github.com/xueqing/ffmpeg-demo/util"

	"github.com/google/logger"
	"github.com/xueqing/goav/libavcodec"
	"github.com/xueqing/goav/libavformat"
	"github.com/xueqing/goav/libavutil"
)

// Muxer mux packets
type Muxer struct {
	pFmtCtx *libavformat.AvFormatContext
}

// New init a muxer
func New() *Muxer {
	m := &Muxer{}
	return m
}

// Close release memory
func (m *Muxer) Close() {
	if m.pFmtCtx != nil {
		// Free an AVFormatContext and all its streams.
		m.pFmtCtx.AvformatFreeContext()
		m.pFmtCtx = nil
	}
}

// Open initialize format context
func (m *Muxer) Open(strURL, strFmt string) (err error) {
	// Allocate an AVFormatContext for an output format. avformat_free_context() can be used to
	// free the context and everything allocated by the framework within it.
	var pFmt *libavformat.AvOutputFormat
	if ret := libavformat.AvformatAllocOutputContext2(&m.pFmtCtx, pFmt, strFmt, strURL); ret < 0 {
		err = fmt.Errorf("Muxer Open: alloc output context error(%v)", libavutil.ErrorFromCode(ret))
		return
	}

	// Create and initialize a AVIOContext for accessing the resource indicated by url.
	var pIOCtx *libavformat.AvIOContext
	if pIOCtx, err = libavformat.AvIOOpen(strURL, libavformat.AvioFlagWrite); err != nil {
		return
	}
	m.pFmtCtx.SetPb(pIOCtx)

	return
}

// AddStream save stream
func (m *Muxer) AddStream(pInStream *libavformat.AvStream) (err error) {
	var (
		pCodec     *libavcodec.AvCodec
		pCodecCtx  *libavcodec.AvCodecContext
		pOutStream *libavformat.AvStream
	)

	codecID := libavcodec.AvCodecID(pInStream.Codec().CodecID())
	// Find a registered encoder with a matching codec ID.
	if pCodec = libavcodec.AvcodecFindEncoder(codecID); pCodec == nil {
		err = fmt.Errorf("Muxer AddStream: find encoder by id(%v) error", libavcodec.AvcodecGetName(codecID))
		return
	}

	// Allocate an AVCodecContext and set its fields to default values. The
	// resulting struct should be freed with avcodec_free_context().
	if pCodecCtx = pCodec.AvcodecAllocContext3(); pCodecCtx == nil {
		err = fmt.Errorf("Muxer AddStream: alloc context error")
		return
	}
	// defer pCodecCtx.AvcodecFreeContext()

	// Add a new stream to a media file.
	pCodecConvert := (*libavformat.AvCodec)(unsafe.Pointer(pCodec))
	if pOutStream = m.pFmtCtx.AvformatNewStream(pCodecConvert); pOutStream == nil {
		err = fmt.Errorf("Muxer AddStream: new stream error")
		return
	}

	if ret := pCodecCtx.AvcodecParametersFromContext(pOutStream.CodecParameters()); ret < 0 {
		err = fmt.Errorf("Muxer AddStream: copy parameters from context error(%v)", libavutil.ErrorFromCode(ret))
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
	case libavformat.AvmediaTypeVideo:
		pOutStream.CodecParameters().SetHeight(pInStream.CodecParameters().Height())
		pOutStream.CodecParameters().SetWidth(pInStream.CodecParameters().Width())
	case libavformat.AvmediaTypeAudio:
		pOutStream.CodecParameters().SetSampleRate(pInStream.CodecParameters().SampleRate())
		pOutStream.CodecParameters().SetChannels(pInStream.CodecParameters().Channels())
		pOutStream.CodecParameters().SetChannelLayout(pInStream.CodecParameters().ChannelLayout())
		pOutStream.CodecParameters().SetFormat(pInStream.CodecParameters().Format())
	default:
		codecTypeConvert := libavutil.AvMediaType(codecType)
		logger.Warningf("Muxer AddStream: unsupported media type(%v)", libavutil.AvGetMediaTypeString(codecTypeConvert))
	}

	return
}

// WriteHeader save stream header
func (m *Muxer) WriteHeader(options map[string]interface{}) (err error) {
	var pDict *libavutil.AvDictionary
	if pDict, err = util.GetAVDictionaryFromMap(options); err != nil {
		return
	}
	defer pDict.AvDictFree()

	// Allocate the stream private data and write the stream header to an output media file.
	if ret := m.pFmtCtx.AvformatWriteHeader((**libavutil.AvDictionary)(unsafe.Pointer(&pDict))); ret < 0 {
		err = fmt.Errorf("Muxer WriteHeader: error(%v)", libavutil.ErrorFromCode(ret))
		return
	}

	return nil
}

// WritePacket mux a packet
func (m *Muxer) WritePacket(pPkt *libavcodec.AvPacket) int {
	// Write a packet to an output media file.
	return m.pFmtCtx.AvWriteFrame(pPkt)
}

// WriteTrailer write stream trailer
func (m *Muxer) WriteTrailer() int {
	// Write the stream trailer to an output media file and free the file private data.
	// May only be called after a successful call to WriteHeader.
	return m.pFmtCtx.AvWriteTrailer()
}

// Streams get streams
func (m *Muxer) Streams() ([]*libavformat.AvStream, error) {
	return m.pFmtCtx.Streams(), nil
}
