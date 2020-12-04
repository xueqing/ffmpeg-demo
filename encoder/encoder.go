package encoder

import (
	"fmt"
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
	return
}
