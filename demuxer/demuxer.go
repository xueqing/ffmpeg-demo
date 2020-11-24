package demuxer

import (
	"fmt"

	"github.com/giorgisio/goav/avcodec"
	"github.com/giorgisio/goav/avformat"
	"github.com/giorgisio/goav/avutil"
)

// Demuxer demux container to get packets
type Demuxer struct {
	pFmtCtx *avformat.Context
}

// New init a demuxer
func New() *Demuxer {
	d := &Demuxer{}
	// Allocate an AVFormatContext.
	// if d.pFmtCtx = avformat.AvformatAllocContext(); d.pFmtCtx == nil {
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
	var pFmt *avformat.InputFormat
	if len(strFmt) != 0 {
		if pFmt = avformat.AvFindInputFormat(strFmt); pFmt == nil {
			err = fmt.Errorf("Demuxer Open: find input format(%v) error", strFmt)
			return
		}
	}

	// Open an input stream and read the header. The codecs are not opened.
	// The stream must be closed with avformat_close_input().
	if ret := avformat.AvformatOpenInput(&d.pFmtCtx, strURL, pFmt, nil); ret < 0 {
		err = fmt.Errorf("Demuxer Open: open input(%v) error(%v)", strURL, avutil.ErrorFromCode(ret))
		return
	}

	// Read packets of a media file to get stream information.
	if ret := d.pFmtCtx.AvformatFindStreamInfo(nil); ret != 0 {
		err = fmt.Errorf("Demuxer Open: failed to faind stream info error(%v)", avutil.ErrorFromCode(ret))
		return
	}

	// Dump information about file onto standard error
	d.pFmtCtx.AvDumpFormat(0, strURL, 0)

	return
}

// Streams get streams
func (d *Demuxer) Streams() ([]*avformat.Stream, error) {
	return d.pFmtCtx.Streams(), nil
}

// ReadPacket get a packet
func (d *Demuxer) ReadPacket(pPkt *avcodec.Packet) int {
	// Return the next frame of a stream.
	return d.pFmtCtx.AvReadFrame(pPkt)
}
