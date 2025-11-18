package nvstream

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/flarexio/game/thirdparty/moonlight"
)

type VideoReader interface {
	moonlight.VideoDecoderRenderer
	io.ReadCloser
}

func NewVideoReader() VideoReader {
	return &videoReader{}
}

type videoReader struct {
	initialWidth  int
	initialHeight int
	videoFormat   int
	refreshRate   int

	stream *bytes.Buffer
	sync.Mutex
}

func (vr *videoReader) Setup(format, width, height, redrawRate int) int {
	fmt.Printf("Setup called with format=%d, %dx%d@%d\n", format, width, height, redrawRate)

	vr.initialWidth = width
	vr.initialHeight = height
	vr.videoFormat = format
	vr.refreshRate = redrawRate

	if rc := vr.initializeDecoder(); rc != 0 {
		return rc
	}

	vr.stream = new(bytes.Buffer)
	return 0
}

func (vr *videoReader) initializeDecoder() int {
	switch {
	case (vr.videoFormat & moonlight.VIDEO_FORMAT_MASK_H264) != 0:
		fmt.Println("Initializing H.264 decoder")

		if vr.initialWidth > 4096 || vr.initialHeight > 4096 {
			fmt.Println("Error: Resolution too high for H.264 decoder")
			return -1
		}

	case (vr.videoFormat & moonlight.VIDEO_FORMAT_MASK_H265) != 0:
		fmt.Println("Initializing H.265 decoder")

	case (vr.videoFormat & moonlight.VIDEO_FORMAT_MASK_AV1) != 0:
		fmt.Println("Initializing AV1 decoder")

	default:
		fmt.Println("Error: Unsupported video format")
		return -3
	}

	return 0
}

func (vr *videoReader) Start() {
	fmt.Println("Start called")
}

func (vr *videoReader) Stop() {
	fmt.Println("Stop called")
}

func (vr *videoReader) Cleanup() {
	fmt.Println("Cleanup called")
}

func (vr *videoReader) SubmitDecodeUnit(decodeUnit *moonlight.DecodeUnit) int {
	for currentEntry := decodeUnit.BufferList; currentEntry != nil; currentEntry = currentEntry.Next {
		length := currentEntry.Length
		if length == 0 {
			continue
		}

		vr.Lock()
		vr.stream.Write(currentEntry.Data[:length])
		vr.Unlock()
	}

	return moonlight.DR_OK
}

func (vr *videoReader) Capabilities() int {
	return 0
}

func (vr *videoReader) Read(p []byte) (n int, err error) {
	vr.Lock()
	defer vr.Unlock()

	if vr.stream.Len() == 0 {
		return 0, io.EOF
	}

	return vr.stream.Read(p)
}

func (vr *videoReader) Close() error {
	return nil
}
