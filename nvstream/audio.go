package nvstream

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/flarexio/game/thirdparty/moonlight"
)

type AudioReader interface {
	moonlight.AudioRenderer
	io.ReadCloser
}

func NewAudioReader() AudioReader {
	return &audioReader{}
}

type audioReader struct {
	stream *bytes.Buffer
	sync.Mutex
}

func (ar *audioReader) Init(audioConfiguration moonlight.AudioConfiguration, opusConfig *moonlight.OpusMultiStreamConfiguration) int {
	switch audioConfiguration.ChannelCount {
	case 2: // Stereo

	case 4: // Quad

	case 6: // 5.1

	case 8: // 7.1

	default:
		fmt.Println("Unsupported channel count:", audioConfiguration.ChannelCount)
		return -1
	}

	bytesPerFrame := audioConfiguration.ChannelCount * 2 * opusConfig.SamplesPerFrame
	fmt.Println("Audio initialized with", audioConfiguration.ChannelCount, "channels,", bytesPerFrame, "bytes per frame")

	ar.stream = new(bytes.Buffer)
	return 0
}

func (ar *audioReader) Start() {
	fmt.Println("Start called")
}

func (ar *audioReader) Stop() {
	fmt.Println("Stop called")
}

func (ar *audioReader) Cleanup() {
	fmt.Println("Cleanup called")
}

func (ar *audioReader) AudioRendererDecodeAndPlaySample(sampleData []byte, sampleLength int) {
	ar.Lock()
	defer ar.Unlock()

	ar.stream.Write(sampleData[:sampleLength])
}

func (ar *audioReader) Capabilities() int {
	return 0
}

func (ar *audioReader) Read(p []byte) (n int, err error) {
	ar.Lock()
	defer ar.Unlock()

	if ar.stream.Len() == 0 {
		return 0, io.EOF
	}

	return ar.stream.Read(p)
}

func (ar *audioReader) Close() error {
	return nil
}
