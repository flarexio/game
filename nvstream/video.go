package nvstream

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
	"unsafe"

	"go.uber.org/zap"

	"github.com/flarexio/game/thirdparty/moonlight"
)

type VideoStream interface {
	moonlight.VideoDecoderRenderer
	io.ReadCloser
}

func NewVideoStream() VideoStream {
	log := zap.L().With(
		zap.String("component", "nvstream.video_stream"),
	)

	return &videoStream{
		log:    log,
		stream: new(bytes.Buffer),
		closed: false,
		cond:   sync.NewCond(&sync.Mutex{}),
	}
}

type videoStream struct {
	log *zap.Logger

	initialWidth  int
	initialHeight int
	videoFormat   int
	refreshRate   int

	stream *bytes.Buffer
	closed bool
	cond   *sync.Cond
	sync.Mutex
}

func (vs *videoStream) Setup(format, width, height, redrawRate int, _ unsafe.Pointer, _ int) int {
	resolution := fmt.Sprintf("%dx%d@%d", width, height, redrawRate)

	log := vs.log.With(
		zap.String("action", "setup"),
		zap.String("resolution", resolution),
		zap.Int("format", format),
	)

	vs.initialWidth = width
	vs.initialHeight = height
	vs.videoFormat = format
	vs.refreshRate = redrawRate

	videoFormat := moonlight.VideoFormatMask(format)

	var mimeType string
	switch {
	case (videoFormat & moonlight.VIDEO_FORMAT_MASK_H264) != 0:
		mimeType = "video/avc"

		if vs.initialWidth > 4096 || vs.initialHeight > 4096 {
			err := errors.New("resolution too high for AVC decoder")
			log.Error("failed to setup video stream", zap.Error(err))
			return -1
		}

	case (videoFormat & moonlight.VIDEO_FORMAT_MASK_H265) != 0:
		mimeType = "video/hevc"

	case (videoFormat & moonlight.VIDEO_FORMAT_MASK_AV1) != 0:
		mimeType = "video/av01"

	default:
		err := errors.New("unsupported video format")
		log.Error("failed to setup video stream", zap.Error(err))
		return -3
	}

	log.Info("setup complete", zap.String("mime", mimeType))
	return 0
}

func (vs *videoStream) Start() {
	vs.log.Info("video stream started", zap.String("action", "start"))
}

func (vs *videoStream) Stop() {
	vs.log.Info("video stream stopped", zap.String("action", "stop"))
}

func (vs *videoStream) Cleanup() {
	vs.Lock()
	vs.stream.Reset()
	vs.Unlock()

	vs.log.Info("video stream cleaned up", zap.String("action", "cleanup"))
}

func (vs *videoStream) SubmitDecodeUnit(decodeUnit *moonlight.DecodeUnit) int {
	vs.Lock()
	defer vs.Unlock()

	isIDR := decodeUnit.FrameType == int(moonlight.FRAME_TYPE_IDR)
	if isIDR {
		if vs.stream.Len() > 0 {
			vs.stream.Reset()
		}

		vs.log.Debug("received IDR frame")
	}

	for currentEntry := decodeUnit.BufferList; currentEntry != nil; currentEntry = currentEntry.Next {
		length := currentEntry.Length
		if length == 0 {
			continue
		}

		vs.stream.Write(currentEntry.Data[:length])
	}

	vs.cond.Signal()

	return moonlight.DR_OK
}

func (vs *videoStream) Capabilities() int {
	vs.log.Info("video stream capabilities requested")
	return 0
}

func (vs *videoStream) Read(p []byte) (n int, err error) {
	vs.cond.L.Lock()
	defer vs.cond.L.Unlock()

	// Wait until there's data to read or the stream is closed
	for vs.stream.Len() == 0 && !vs.closed {
		vs.cond.Wait()
	}

	if vs.closed && vs.stream.Len() == 0 {
		return 0, io.EOF
	}

	return vs.stream.Read(p)
}

func (vs *videoStream) Close() error {
	vs.Lock()
	vs.closed = true
	vs.stream.Reset()
	vs.Unlock()

	vs.cond.Broadcast()

	vs.log.Info("video stream closed", zap.String("action", "close"))
	return nil
}
