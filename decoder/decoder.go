package decoder

import (
	"fmt"
	"unsafe"

	"github.com/google/logger"
	"github.com/xueqing/goav/libavcodec"
	"github.com/xueqing/goav/libavformat"
	"github.com/xueqing/goav/libavutil"
)

// Decoder decode AVPacket to AVFrame
type Decoder struct {
	// must call libavutil.AvFrameFree(pFrame) after use
	FrameHandler func(pFrame *libavutil.AvFrame) (err error)

	pInFmtCtx *libavformat.AvFormatContext
	pDecCtx   *libavcodec.AvCodecContext
	mediaType libavutil.AvMediaType
	streamIdx int
}

// New create a Decoder
func New(pInFmtCtx *libavformat.AvFormatContext) *Decoder {
	return &Decoder{
		pInFmtCtx: pInFmtCtx,
		mediaType: libavutil.AvmediaTypeUnknown,
	}
}

// DecCodecContext ...
func (d *Decoder) DecCodecContext() *libavcodec.AvCodecContext {
	return d.pDecCtx
}

// StreamIdx Return streamIdx
func (d *Decoder) StreamIdx() int {
	return d.streamIdx
}

// Close ...
func (d *Decoder) Close() {
	if d.pDecCtx != nil {
		d.pDecCtx.AvcodecFreeContext()
		d.pDecCtx = nil
	}
}

// Open set decoder
func (d *Decoder) Open(pInStream *libavformat.AvStream) (err error) {
	var (
		pDec *libavcodec.AvCodec
	)

	if d.pDecCtx != nil {
		err = fmt.Errorf("Decoder Open: codec context is not nil")
		return
	}

	codecID := pInStream.CodecParameters().CodecID()
	// Find a registered decoder with a matching codec ID.
	pDec = libavcodec.AvcodecFindDecoder(codecID)
	if pDec == nil {
		err = fmt.Errorf("Decoder Open: find decoder by id(%v) error", libavcodec.AvcodecGetName(codecID))
		return
	}

	// Allocate an AVCodecContext and set its fields to default values. The
	// resulting struct should be freed with avcodec_free_context().
	if d.pDecCtx = pDec.AvcodecAllocContext3(); d.pDecCtx == nil {
		err = fmt.Errorf("Decoder Open: alloc context error")
		return
	}

	// copy decoder parameters to decoder context
	if ret := d.pDecCtx.AvcodecParametersToContext(pInStream.CodecParameters()); ret < 0 {
		err = fmt.Errorf("Decoder Open: copy decoder parameters to decoder context error(%v)", libavutil.ErrorFromCode(ret))
		return
	}

	// open decoder
	d.mediaType = libavutil.AvMediaType(pInStream.CodecParameters().CodecType())
	if d.mediaType == libavutil.AvmediaTypeVideo {
		d.pDecCtx.SetFramerate(d.pInFmtCtx.AvGuessFrameRate(pInStream, nil))
	}
	if ret := d.pDecCtx.AvcodecOpen2(pDec, nil); ret < 0 {
		err = fmt.Errorf("Decoder Open: open decoder error(%v)", libavutil.ErrorFromCode(ret))
		return
	}

	d.streamIdx = pInStream.Index()
	return
}

// DecodePacket Decode packet to frame
func (d *Decoder) DecodePacket(pPkt *libavcodec.AvPacket) (pFrame *libavutil.AvFrame, gotFrame int, err error) {
	var (
		ret int
	)

	if d.pDecCtx == nil {
		err = fmt.Errorf("Decoder DecodePacket: codec context is nil")
		return
	}

	if pFrame = libavutil.AvFrameAlloc(); pFrame == nil {
		err = fmt.Errorf("Decoder DecodePacket: failed to alloc memory for frame")
		return
	}

	pFrameConvert := (*libavcodec.AvFrame)(unsafe.Pointer(pFrame))
	if d.mediaType == libavutil.AvmediaTypeVideo {
		ret = d.pDecCtx.AvcodecDecodeVideo2(pFrameConvert, &gotFrame, pPkt)
	} else if d.mediaType == libavutil.AvmediaTypeAudio {
		ret = d.pDecCtx.AvcodecDecodeAudio4(pFrameConvert, &gotFrame, pPkt)
	} else {
		err = fmt.Errorf("Decoder DecodePacket: unsupported mediaType(%v)", libavutil.AvGetMediaTypeString(d.mediaType))
		libavutil.AvFrameFree(pFrame)
		return
	}
	if ret < 0 {
		err = fmt.Errorf("Decoder DecodePacket: error(%v)", libavutil.ErrorFromCode(ret))
		libavutil.AvFrameFree(pFrame)
		return
	}
	if gotFrame == 1 {
		pFrame.SetPts(pFrame.BestEffortTimestamp())
	}
	return
}

// Send Supply raw packet data as input to a decoder.
func (d *Decoder) Send(pPkt *libavcodec.AvPacket) (err error) {
	if d.pDecCtx == nil {
		err = fmt.Errorf("Decoder Send: codec context is nil")
		return
	}
	/*
	 * @return 0 on success, otherwise negative error code:
	 *      AVERROR(EAGAIN):   input is not accepted in the current state - user
	 *                         must read output with avcodec_receive_frame() (once
	 *                         all output is read, the packet should be resent, and
	 *                         the call will not fail with EAGAIN).
	 *      AVERROR_EOF:       the decoder has been flushed, and no new packets can
	 *                         be sent to it (also returned if more than 1 flush
	 *                         packet is sent)
	 *      AVERROR(EINVAL):   codec not opened, it is an encoder, or requires flush
	 *      AVERROR(ENOMEM):   failed to add packet to internal queue, or similar
	 *      other errors: legitimate decoding errors
	 */
	if ret := d.pDecCtx.AvcodecSendPacket(pPkt); ret < 0 {
		err = fmt.Errorf("Decoder Send: error(%v)", libavutil.ErrorFromCode(ret))
		return
	}
	return
}

// Receive Return decoded output data from a decoder.
func (d *Decoder) Receive() (pFrame *libavutil.AvFrame, err error) {
	if d.pDecCtx == nil {
		err = fmt.Errorf("Decoder Receive: codec context is nil")
		return
	}

	if pFrame = libavutil.AvFrameAlloc(); pFrame == nil {
		err = fmt.Errorf("Decoder Receive: failed to alloc memory for frame")
		return
	}

	pFrameConvert := (*libavcodec.AvFrame)(unsafe.Pointer(pFrame))
	/*
	 * @return
	 *      0:                 success, a frame was returned
	 *      AVERROR(EAGAIN):   output is not available in this state - user must try
	 *                         to send new input
	 *      AVERROR_EOF:       the decoder has been fully flushed, and there will be
	 *                         no more output frames
	 *      AVERROR(EINVAL):   codec not opened, or it is an encoder
	 *      other negative values: legitimate decoding errors
	 */
	ret := d.pDecCtx.AvcodecReceiveFrame(pFrameConvert)
	if ret == libavutil.AvErrorEAGAIN || ret == libavutil.AvErrorEOF {
		goto end
	}
	if ret < 0 {
		err = fmt.Errorf("Decoder Receive: error(%v)", libavutil.ErrorFromCode(ret))
		goto end
	}
	return

end:
	libavutil.AvFrameFree(pFrame)
	pFrame = nil
	return
}

// Decode Decode packet to frame
func (d *Decoder) Decode(pPkt *libavcodec.AvPacket) (err error) {
	if err = d.Send(pPkt); err != nil {
		logger.Errorf("Decoder Decode: Send error(%v)", err)
		return
	}

	var pFrame *libavutil.AvFrame
	for {
		pFrame, err = d.Receive()
		if err != nil {
			logger.Errorf("Decoder Decode: Receive error(%v)", err)
			return
		}
		if pFrame == nil {
			return
		}
		if d.FrameHandler != nil {
			if err = d.FrameHandler(pFrame); err != nil {
				return
			}
		}
	}
}
