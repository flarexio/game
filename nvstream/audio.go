package nvstream

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"go.uber.org/zap"

	"github.com/flarexio/game/thirdparty/moonlight"
)

type AudioStream interface {
	moonlight.AudioRenderer
	io.ReadCloser
}

func NewAudioStream() AudioStream {
	log := zap.L().With(
		zap.String("component", "nvstream.audio_stream"),
		zap.String("mime", "audio/opus"),
	)

	return &audioStream{
		log:    log,
		stream: new(bytes.Buffer),
	}
}

type audioStream struct {
	log *zap.Logger

	stream *bytes.Buffer
	sync.Mutex
}

func (as *audioStream) Init(audioConfiguration moonlight.AudioConfiguration, opusConfig *moonlight.OpusMultiStreamConfiguration) int {
	log := as.log.With(
		zap.String("action", "init"),
		zap.Int("channel_count", audioConfiguration.ChannelCount),
		zap.Int("samples_per_frame", opusConfig.SamplesPerFrame),
	)

	var layout string
	switch audioConfiguration.ChannelCount {
	case 2: // Stereo
		layout = "stereo"

	case 1: // Mono
		layout = "mono"

	case 4: // Quad
		layout = "quad"

	case 6: // 5.1
		layout = "5.1"

	case 8: // 7.1
		layout = "7.1"

	default:
		err := fmt.Errorf("unsupported channel count: %d", audioConfiguration.ChannelCount)
		log.Error("failed to initialize audio stream", zap.Error(err))
		return -1
	}

	bytesPerFrame := audioConfiguration.ChannelCount * 2 * opusConfig.SamplesPerFrame

	log.Info("audio stream initialized successfully",
		zap.String("layout", layout),
		zap.Int("bytes_per_frame", bytesPerFrame))

	return 0
}

func (as *audioStream) Start() {
	as.log.Info("audio stream started")
}

func (as *audioStream) Stop() {
	as.log.Info("audio stream stopped")
}

func (as *audioStream) Cleanup() {
	as.log.Info("audio stream cleaned up")
}

func (as *audioStream) AudioRendererDecodeAndPlaySample(sampleData []byte, sampleLength int) {
	as.Lock()
	defer as.Unlock()

	as.stream.Write(sampleData[:sampleLength])
}

func (as *audioStream) Capabilities() int {
	as.log.Info("audio stream capabilities requested")
	return 0
}

func (as *audioStream) Read(p []byte) (n int, err error) {
	as.Lock()
	defer as.Unlock()

	if as.stream.Len() == 0 {
		return 0, io.EOF
	}

	return as.stream.Read(p)
}

func (as *audioStream) Close() error {
	as.log.Info("audio stream closed", zap.String("action", "close"))
	return nil
}
