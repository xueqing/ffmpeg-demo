package encoder

import (
	"fmt"
	"unsafe"

	"github.com/xueqing/goav/libavcodec"
	"github.com/xueqing/goav/libavformat"
)

// Encoder encode AVFrame to AVPacket
type Encoder struct {
	pEncCtx *libavcodec.AvCodecContext
	pEnc    *libavcodec.AvCodec
}

// New create a Encoder
func New() *Encoder {
	return &Encoder{}
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
		err = fmt.Errorf("setStreamContext: find encoder by id(%v) error", libavcodec.AvcodecGetName(codecID))
		return
	}
	// Allocate an AVCodecContext and set its fields to default values. The
	// resulting struct should be freed with avcodec_free_context().
	if e.pEncCtx = e.pEnc.AvcodecAllocContext3(); e.pEncCtx == nil {
		err = fmt.Errorf("setStreamContext: alloc encoder context error")
		return
	}
	return
}
