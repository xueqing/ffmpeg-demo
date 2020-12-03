package demuxer

import (
	"fmt"

	"github.com/google/logger"

	"github.com/xueqing/goav/libavcodec"
	"github.com/xueqing/goav/libavformat"
	"github.com/xueqing/goav/libavutil"
)

// Demuxer demux container to get packets
type Demuxer struct {
	pInFmtCtx *libavformat.AvFormatContext
}

// New init a demuxer
func New() *Demuxer {
	return &Demuxer{}
}

// InFormatContext ...
func (d *Demuxer) InFormatContext() *libavformat.AvFormatContext {
	return d.pInFmtCtx
}

// Close release some memory
func (d *Demuxer) Close() {
	// Close an opened input AVFormatContext. Free it and all its contents
	if d.pInFmtCtx != nil {
		d.pInFmtCtx.AvformatCloseInput()
		d.pInFmtCtx = nil
	}
}

// Open initlize format context
func (d *Demuxer) Open(strURL, strFmt string) (err error) {
	if d.pInFmtCtx != nil {
		err = fmt.Errorf("Demuxer Open: input format context is not nil")
		return
	}

	// Find AVInputFormat based on the short name of the input format.
	var pInFmt *libavformat.AvInputFormat
	if len(strFmt) != 0 {
		if pInFmt = libavformat.AvFindInputFormat(strFmt); pInFmt == nil {
			err = fmt.Errorf("Demuxer Open: find input format(%v) error", strFmt)
			return
		}
	}

	// Open an input stream and read the header. The codecs are not opened.
	// The stream must be closed with avformat_close_input().
	if ret := libavformat.AvformatOpenInput(&d.pInFmtCtx, strURL, pInFmt, nil); ret < 0 {
		err = fmt.Errorf("Demuxer Open: open input(%v) error(%v)", strURL, libavutil.ErrorFromCode(ret))
		return
	}

	// Read packets of a media file to get stream information.
	if ret := d.pInFmtCtx.AvformatFindStreamInfo(nil); ret != 0 {
		err = fmt.Errorf("Demuxer Open: failed to faind stream info error(%v)", libavutil.ErrorFromCode(ret))
		return
	}

	// Dump information about file onto standard error
	d.pInFmtCtx.AvDumpFormat(0, strURL, 0)

	return
}

// Streams get streams
func (d *Demuxer) Streams() ([]*libavformat.AvStream, error) {
	if d.pInFmtCtx == nil {
		return nil, fmt.Errorf("Demuxer Streams: input format context is nil")
	}
	return d.pInFmtCtx.Streams(), nil
}

// ReadPacket get a packet
func (d *Demuxer) ReadPacket(pPkt *libavcodec.AvPacket) int {
	if d.pInFmtCtx == nil {
		logger.Errorf("Demuxer ReadPacket: input format context is nil")
		return -1
	}
	// Return the next frame of a stream.
	return d.pInFmtCtx.AvReadFrame(pPkt)
}
