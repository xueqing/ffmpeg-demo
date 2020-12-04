package encoder

import (
	"fmt"
	"io"
	"unsafe"

	"github.com/xueqing/goav/libavcodec"
	"github.com/xueqing/goav/libavformat"
	"github.com/xueqing/goav/libavutil"
)

// Encoder encode AVFrame to AVPacket
type Encoder struct {
	pEncCtx   *libavcodec.AvCodecContext
	pEnc      *libavcodec.AvCodec
	mediaType libavutil.AvMediaType
	streamIdx int
}

// New create a Encoder
func New() *Encoder {
	return &Encoder{
		mediaType: libavutil.AvmediaTypeUnknown,
	}
}

// EncCodecContext ...
func (e *Encoder) EncCodecContext() *libavcodec.AvCodecContext {
	return e.pEncCtx
}

// EncCodec ...
func (e *Encoder) EncCodec() *libavcodec.AvCodec {
	return e.pEnc
}

// StreamIdx Return streamIdx
func (e *Encoder) StreamIdx() int {
	return e.streamIdx
}

// Close ...
func (e *Encoder) Close() {

}

// Open ...
func (e *Encoder) Open(pInStream *libavformat.AvStream) (err error) {
	pDecCtx := (*libavcodec.AvCodecContext)(unsafe.Pointer(pInStream.Codec()))
	codecID := libavcodec.AvCodecID(pDecCtx.CodecID())
	// Find a registered encoder with a matching codec ID.
	// in this example, we choose transcoding to same codec
	if e.pEnc = libavcodec.AvcodecFindEncoder(codecID); e.pEnc == nil {
		err = fmt.Errorf("Encoder Open: find encoder by id(%v) error", libavcodec.AvcodecGetName(codecID))
		return
	}
	// Allocate an AVCodecContext and set its fields to default values. The
	// resulting struct should be freed with avcodec_free_context().
	if e.pEncCtx = e.pEnc.AvcodecAllocContext3(); e.pEncCtx == nil {
		err = fmt.Errorf("Encoder Open: alloc encoder context error")
		return
	}
	e.mediaType = libavutil.AvMediaType(pInStream.CodecParameters().CodecType())
	e.streamIdx = pInStream.Index()
	return
}

// EncodeFrame Encode frame to packet
func (e *Encoder) EncodeFrame(pFrame *libavcodec.AvFrame, gotFrame *int) (pEncPkt *libavcodec.AvPacket, err error) {
	var (
		ret, localGotFrame int
	)

	if gotFrame == nil {
		gotFrame = &localGotFrame
	}

	if pEncPkt = libavcodec.AvPacketAlloc(); pEncPkt == nil {
		err = fmt.Errorf("Encoder EncodeFrame: alloc packet error")
		return
	}
	pEncPkt.AvInitPacket()

	if e.mediaType == libavutil.AvmediaTypeVideo {
		ret = e.pEncCtx.AvcodecEncodeVideo2(pEncPkt, pFrame, gotFrame)
	} else if e.mediaType == libavutil.AvmediaTypeAudio {
		ret = e.pEncCtx.AvcodecEncodeAudio2(pEncPkt, pFrame, gotFrame)
	} else {
		err = fmt.Errorf("Encoder EncodeFrame: unsupported mediaType(%v)", libavutil.AvGetMediaTypeString(e.mediaType))
		pEncPkt.AvPacketUnref()
		return
	}
	if ret < 0 {
		err = fmt.Errorf("Encoder EncodeFrame: error(%v)", libavutil.ErrorFromCode(ret))
		pEncPkt.AvPacketUnref()
		return
	}
	if (*gotFrame) == 0 {
		pEncPkt.AvPacketUnref()
		return nil, nil
	}
	pEncPkt.SetStreamIndex(e.streamIdx)
	return
}

// Send Supply a raw video or audio frame to the encoder.
// Return io.EOF when ffmpeg return AVERROR(EAGAIN)/AVERROR_EOF
func (e *Encoder) Send(pFrame *libavutil.AvFrame) (err error) {
	if e.pEncCtx == nil {
		err = fmt.Errorf("Encoder Send: codec context is nil")
		return
	}

	pFrameConvert := (*libavcodec.AvFrame)(unsafe.Pointer(pFrame))
	/*
	 * @return 0 on success, otherwise negative error code:
	 *      AVERROR(EAGAIN):   input is not accepted in the current state - user
	 *                         must read output with avcodec_receive_packet() (once
	 *                         all output is read, the packet should be resent, and
	 *                         the call will not fail with EAGAIN).
	 *      AVERROR_EOF:       the encoder has been flushed, and no new frames can
	 *                         be sent to it
	 *      AVERROR(EINVAL):   codec not opened, refcounted_frames not set, it is a
	 *                         decoder, or requires flush
	 *      AVERROR(ENOMEM):   failed to add packet to internal queue, or similar
	 *      other errors: legitimate decoding errors
	 */
	if ret := e.pEncCtx.AvcodecSendFrame(pFrameConvert); ret < 0 {
		if ret == libavutil.AvErrorEOF || ret == libavutil.AvErrorEAGAIN {
			err = io.EOF
		} else {
			err = fmt.Errorf("Encoder Send: error(%v)", libavutil.ErrorFromCode(ret))
		}
		return
	}
	return
}

// Receive Read encoded data from the encoder.
// Return io.EOF when ffmpeg return AVERROR(EAGAIN)/AVERROR_EOF
func (e *Encoder) Receive() (pPkt *libavcodec.AvPacket, err error) {
	if e.pEncCtx == nil {
		err = fmt.Errorf("Encoder Receive: codec context is nil")
		return
	}

	if pPkt = libavcodec.AvPacketAlloc(); pPkt == nil {
		err = fmt.Errorf("Encoder Receive: alloc packet error")
		return
	}
	pPkt.AvInitPacket()

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
	ret := e.pEncCtx.AvcodecReceivePacket(pPkt)
	if ret == libavutil.AvErrorEAGAIN || ret == libavutil.AvErrorEOF {
		err = io.EOF
		goto end
	}
	if ret < 0 {
		err = fmt.Errorf("Encoder Receive: error(%v)", libavutil.ErrorFromCode(ret))
		goto end
	}
	pPkt.SetStreamIndex(e.streamIdx)
	return

end:
	pPkt.AvPacketUnref()
	pPkt = nil
	return
}
