package demuxer

import (
	"fmt"

	"github.com/xueqing/goav/libavcodec"
	"github.com/xueqing/goav/libavformat"
	"github.com/xueqing/goav/libavutil"
)

// Demuxer demux container to get packets
type Demuxer struct {
	pFmtCtx *libavformat.AvFormatContext
}

// New init a demuxer
func New() *Demuxer {
	d := &Demuxer{}
	// Allocate an AVFormatContext.
	// if d.pFmtCtx = libavformat.AvformatAllocContext(); d.pFmtCtx == nil {
	// 	logger.Warningf("Demuxer New: failed to alloc context")
	// 	return nil
	// }
	return d
}

// Close release some memory
func (d *Demuxer) Close() {
	// Close an opened input AVFormatContext. Free it and all its contents
	d.pFmtCtx.AvformatCloseInput()
	d.pFmtCtx = nil
}

// Open initlize format context
func (d *Demuxer) Open(strURL, strFmt string) (err error) {
	// Find AVInputFormat based on the short name of the input format.
	var pFmt *libavformat.AvInputFormat
	if len(strFmt) != 0 {
		if pFmt = libavformat.AvFindInputFormat(strFmt); pFmt == nil {
			err = fmt.Errorf("Demuxer Open: find input format(%v) error", strFmt)
			return
		}
	}

	// Open an input stream and read the header. The codecs are not opened.
	// The stream must be closed with avformat_close_input().
	if ret := libavformat.AvformatOpenInput(&d.pFmtCtx, strURL, pFmt, nil); ret < 0 {
		err = fmt.Errorf("Demuxer Open: open input(%v) error(%v)", strURL, libavutil.ErrorFromCode(ret))
		return
	}

	// Read packets of a media file to get stream information.
	if ret := d.pFmtCtx.AvformatFindStreamInfo(nil); ret != 0 {
		err = fmt.Errorf("Demuxer Open: failed to faind stream info error(%v)", libavutil.ErrorFromCode(ret))
		return
	}

	// Dump information about file onto standard error
	d.pFmtCtx.AvDumpFormat(0, strURL, 0)

	return
}

// Streams get streams
func (d *Demuxer) Streams() ([]*libavformat.AvStream, error) {
	return d.pFmtCtx.Streams(), nil
}

// ReadPacket get a packet
func (d *Demuxer) ReadPacket(pPkt *libavcodec.AvPacket) int {
	// Return the next frame of a stream.
	return d.pFmtCtx.AvReadFrame(pPkt)
}
