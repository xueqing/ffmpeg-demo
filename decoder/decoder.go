package decoder

import (
	"fmt"

	"github.com/xueqing/goav/libavcodec"
	"github.com/xueqing/goav/libavformat"
	"github.com/xueqing/goav/libavutil"
)

// Decoder decode AVPacket to AVFrame
type Decoder struct {
	pInFmtCtx *libavformat.AvFormatContext
	pDecCtx   *libavcodec.AvCodecContext
}

// New create a Decoder
func New(pInFmtCtx *libavformat.AvFormatContext) *Decoder {
	return &Decoder{
		pInFmtCtx: pInFmtCtx,
	}
}

// DecCodecContext ...
func (d *Decoder) DecCodecContext() *libavcodec.AvCodecContext {
	return d.pDecCtx
}

// Close ...
func (d *Decoder) Close() {
	d.pDecCtx.AvcodecFreeContext()
}

// Open set decoder
func (d *Decoder) Open(pInStream *libavformat.AvStream) (err error) {
	var (
		pDec *libavcodec.AvCodec
	)

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
	if pInStream.CodecParameters().CodecType() == libavformat.AvmediaTypeVideo {
		d.pDecCtx.SetFramerate(d.pInFmtCtx.AvGuessFrameRate(pInStream, nil))
	}
	if ret := d.pDecCtx.AvcodecOpen2(pDec, nil); ret < 0 {
		err = fmt.Errorf("Decoder Open: open decoder error(%v)", libavutil.ErrorFromCode(ret))
		return
	}

	return
}
