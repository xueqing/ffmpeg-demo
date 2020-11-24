package muxer

import (
	"fmt"
	"unsafe"

	"github.com/xueqing/ffmpeg-demo/util"

	"github.com/giorgisio/goav/avcodec"
	"github.com/giorgisio/goav/avformat"
	"github.com/giorgisio/goav/avutil"
	"github.com/google/logger"
)

// Muxer mux packets
type Muxer struct {
	pFmtCtx *avformat.Context
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
	var pFmt *avformat.OutputFormat
	if ret := avformat.AvformatAllocOutputContext2(&m.pFmtCtx, pFmt, strFmt, strURL); ret < 0 {
		err = fmt.Errorf("Muxer Open: alloc output context error(%v)", avutil.ErrorFromCode(ret))
		return
	}

	// Create and initialize a AVIOContext for accessing the resource indicated by url.
	var pIOCtx *avformat.AvIOContext
	if pIOCtx, err = avformat.AvIOOpen(strURL, avformat.AVIO_FLAG_WRITE); err != nil {
		return
	}
	m.pFmtCtx.SetPb(pIOCtx)

	return
}

// AddStream save stream
func (m *Muxer) AddStream(pInStream *avformat.Stream) (err error) {
	var (
		pCodec     *avcodec.Codec
		pCodecCtx  *avcodec.Context
		pOutStream *avformat.Stream
	)

	codecID := avcodec.CodecId(pInStream.Codec().GetCodecId())
	// Find a registered encoder with a matching codec ID.
	if pCodec = avcodec.AvcodecFindEncoder(codecID); pCodec == nil {
		err = fmt.Errorf("Muxer AddStream: find encoder by id(%v) error", avcodec.AvcodecGetName(codecID))
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
	pCodecConvert := (*avformat.AvCodec)(unsafe.Pointer(pCodec))
	if pOutStream = m.pFmtCtx.AvformatNewStream(pCodecConvert); pOutStream == nil {
		err = fmt.Errorf("Muxer AddStream: new stream error")
		return
	}

	if ret := pCodecCtx.AvcodecParametersFromContext(pOutStream.CodecParameters()); ret < 0 {
		err = fmt.Errorf("Muxer AddStream: copy parameters from context error(%v)", avutil.ErrorFromCode(ret))
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

	switch codecType := pInStream.CodecParameters().AvCodecGetType(); codecType {
	case avformat.AVMEDIA_TYPE_VIDEO:
		pOutStream.CodecParameters().AvCodecSetHeight(pInStream.CodecParameters().AvCodecGetHeight())
		pOutStream.CodecParameters().AvCodecSetWidth(pInStream.CodecParameters().AvCodecGetWidth())
	case avformat.AVMEDIA_TYPE_AUDIO:
		pOutStream.CodecParameters().AvCodecSetSampleRate(pInStream.CodecParameters().AvCodecGetSampleRate())
		pOutStream.CodecParameters().AvCodecSetChannels(pInStream.CodecParameters().AvCodecGetChannels())
		pOutStream.CodecParameters().AvCodecSetChannelLayout(pInStream.CodecParameters().AvCodecGetChannelLayout())
		pOutStream.CodecParameters().AvCodecSetFormat(pInStream.CodecParameters().AvCodecGetFormat())
	default:
		codecTypeConvert := avutil.MediaType(codecType)
		logger.Warningf("Muxer AddStream: unsupported media type(%v)", avutil.AvGetMediaTypeString(codecTypeConvert))
	}

	return
}

// WriteHeader save stream header
func (m *Muxer) WriteHeader(options map[string]interface{}) (err error) {
	var pDict *avutil.Dictionary
	if pDict, err = util.GetAVDictionaryFromMap(options); err != nil {
		return
	}
	defer pDict.AvDictFree()

	// Allocate the stream private data and write the stream header to an output media file.
	if ret := m.pFmtCtx.AvformatWriteHeader((**avutil.Dictionary)(unsafe.Pointer(&pDict))); ret < 0 {
		err = fmt.Errorf("Muxer WriteHeader: error(%v)", avutil.ErrorFromCode(ret))
		return
	}

	return nil
}

// WritePacket mux a packet
func (m *Muxer) WritePacket(pPkt *avcodec.Packet) int {
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
func (m *Muxer) Streams() ([]*avformat.Stream, error) {
	return m.pFmtCtx.Streams(), nil
}
