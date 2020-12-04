package muxer

import (
	"fmt"
	"unsafe"

	"github.com/google/logger"

	"github.com/xueqing/ffmpeg-demo/util"
	"github.com/xueqing/goav/libavcodec"
	"github.com/xueqing/goav/libavformat"
	"github.com/xueqing/goav/libavutil"
)

// Muxer mux packets
type Muxer struct {
	pOutFmtCtx *libavformat.AvFormatContext
}

// New init a muxer
func New() *Muxer {
	m := &Muxer{}
	return m
}

// OutFormatContext ...
func (m *Muxer) OutFormatContext() *libavformat.AvFormatContext {
	return m.pOutFmtCtx
}

// Close release memory
func (m *Muxer) Close() {
	if m.pOutFmtCtx != nil {
		if (m.pOutFmtCtx.Flags() & libavformat.AvfmtNofile) == 0 {
			libavformat.AvioClosep(m.pOutFmtCtx.Pb())
		}
		// Free an AVFormatContext and all its streams.
		m.pOutFmtCtx.AvformatFreeContext()
		m.pOutFmtCtx = nil
	}
}

// Open initialize format context
func (m *Muxer) Open(strURL, strFmt string) (err error) {
	if m.pOutFmtCtx != nil {
		err = fmt.Errorf("Muxer Open: output format context is not nil")
		return
	}

	// Allocate an AVFormatContext for an output format. avformat_free_context() can be used to
	// free the context and everything allocated by the framework within it.
	var pOutFmt *libavformat.AvOutputFormat
	if ret := libavformat.AvformatAllocOutputContext2(&m.pOutFmtCtx, pOutFmt, strFmt, strURL); ret < 0 {
		err = fmt.Errorf("Muxer Open: alloc output context error(%v)", libavutil.ErrorFromCode(ret))
		return
	}

	// Create and initialize a AVIOContext for accessing the resource indicated by url.
	if (m.pOutFmtCtx.Flags() & libavformat.AvfmtNofile) == 0 {
		var pIOCtx *libavformat.AvIOContext
		if pIOCtx, err = libavformat.AvIOOpen(strURL, libavformat.AvioFlagWrite); err != nil {
			return
		}
		m.pOutFmtCtx.SetPb(pIOCtx)
	}

	return
}

// AddStream save stream
func (m *Muxer) AddStream(pInStream *libavformat.AvStream) (pOutStream *libavformat.AvStream, err error) {
	if m.pOutFmtCtx == nil {
		err = fmt.Errorf("Muxer AddStream: output format context is nil")
		return
	}
	if pOutStream = m.pOutFmtCtx.AvformatNewStream(nil); pOutStream == nil {
		err = fmt.Errorf("Muxer AddStream: new stream error")
		return
	}
	return
}

// WriteHeader save stream header
func (m *Muxer) WriteHeader(options map[string]interface{}) (err error) {
	if m.pOutFmtCtx == nil {
		err = fmt.Errorf("Muxer WriteHeader: output format context is nil")
		return
	}
	var pDict *libavutil.AvDictionary
	if pDict, err = util.GetAVDictionaryFromMap(options); err != nil {
		return
	}
	defer pDict.AvDictFree()

	// Allocate the stream private data and write the stream header to an output media file.
	if ret := m.pOutFmtCtx.AvformatWriteHeader((**libavutil.AvDictionary)(unsafe.Pointer(&pDict))); ret < 0 {
		err = fmt.Errorf("Muxer WriteHeader: error(%v)", libavutil.ErrorFromCode(ret))
		return
	}

	return nil
}

// WritePacket mux a packet
func (m *Muxer) WritePacket(pPkt *libavcodec.AvPacket) (err error) {
	if m.pOutFmtCtx == nil {
		err = fmt.Errorf("Muxer WritePacket: output format context is nil")
		return
	}
	// Write a packet to an output media file.
	if ret := m.pOutFmtCtx.AvWriteFrame(pPkt); ret < 0 {
		err = fmt.Errorf("Muxer WritePacket: Write frame error(%v)", libavutil.ErrorFromCode(ret))
		return
	}
	return
}

// IntervedWritePacket mux a packet
func (m *Muxer) IntervedWritePacket(pPkt *libavcodec.AvPacket) (err error) {
	if m.pOutFmtCtx == nil {
		err = fmt.Errorf("Muxer IntervedWritePacket: output format context is nil")
		return
	}
	// Write a packet to an output media file.
	if ret := m.pOutFmtCtx.AvInterleavedWriteFrame(pPkt); ret < 0 {
		err = fmt.Errorf("Muxer IntervedWritePacket: Write frame error(%v)", libavutil.ErrorFromCode(ret))
		return
	}
	return
}

// WriteTrailer write stream trailer
func (m *Muxer) WriteTrailer() int {
	if m.pOutFmtCtx == nil {
		logger.Errorf("Muxer WriteTrailer: output format context is nil")
		return -1
	}
	// Write the stream trailer to an output media file and free the file private data.
	// May only be called after a successful call to WriteHeader.
	return m.pOutFmtCtx.AvWriteTrailer()
}

// Streams get streams
func (m *Muxer) Streams() ([]*libavformat.AvStream, error) {
	if m.pOutFmtCtx == nil {
		return nil, fmt.Errorf("Muxer Streams: output format context is nil")
	}
	return m.pOutFmtCtx.Streams(), nil
}
