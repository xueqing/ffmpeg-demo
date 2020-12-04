package decoder

import (
	"fmt"
	"unsafe"

	"github.com/xueqing/goav/libavcodec"
	"github.com/xueqing/goav/libavformat"
	"github.com/xueqing/goav/libavutil"
)

// Decoder decode AVPacket to AVFrame
type Decoder struct {
	pInFmtCtx *libavformat.AvFormatContext
	pDecCtx   *libavcodec.AvCodecContext
	mediaType libavutil.AvMediaType
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
