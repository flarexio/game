package nvstream

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"
	"unsafe"

	"go.uber.org/zap"

	"github.com/flarexio/game/thirdparty/moonlight"
)

type AudioStream interface {
	moonlight.AudioRenderer
	io.ReadCloser
	SampleDuration() time.Duration
}

func NewAudioStream() AudioStream {
	log := zap.L().With(
		zap.String("component", "nvstream.audio_stream"),
		zap.String("mime", "audio/opus"),
	)

	return &audioStream{
		log:    log,
		stream: new(bytes.Buffer),
		closed: false,
		cond:   sync.NewCond(&sync.Mutex{}),
	}
}

type audioStream struct {
	log *zap.Logger

	sampleDuration time.Duration
	stream         *bytes.Buffer
	closed         bool
	cond           *sync.Cond
	sync.Mutex
}

func (as *audioStream) Init(audioConfiguration moonlight.AudioConfiguration, opusConfig *moonlight.OpusMultiStreamConfiguration, _ unsafe.Pointer, _ int) int {
	log := as.log.With(
		zap.String("action", "init"),
		zap.Int("channel_count", audioConfiguration.ChannelCount),
		zap.Int("samples_per_frame", opusConfig.SamplesPerFrame),
	)

	var layout string
	switch audioConfiguration.ChannelCount {
	case 1: // Mono
		layout = "mono"

	case 2: // Stereo
		layout = "stereo"

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

	sampleRate := opusConfig.SampleRate
	durationMs := (opusConfig.SamplesPerFrame * 1000) / sampleRate

	as.Lock()
	as.sampleDuration = time.Duration(durationMs) * time.Millisecond
	as.Unlock()

	log.Info("audio stream initialized successfully",
		zap.String("layout", layout),
		zap.Int("sample_rate", sampleRate),
		zap.Duration("duration", as.sampleDuration),
	)

	return 0
}

func (as *audioStream) SampleDuration() time.Duration {
	as.Lock()
	duration := as.sampleDuration
	as.Unlock()
	return duration
}

func (as *audioStream) Start() {
	as.log.Info("audio stream started", zap.String("action", "start"))
}

func (as *audioStream) Stop() {
	as.log.Info("audio stream stopped", zap.String("action", "stop"))
}

func (as *audioStream) Cleanup() {
	as.Lock()
	as.stream.Reset()
	as.Unlock()

	as.log.Info("audio stream cleaned up", zap.String("action", "cleanup"))
}

func (as *audioStream) PlayEncodedSample(sampleData []byte, sampleLength int) {
	as.Lock()
	defer as.Unlock()

	if as.closed || sampleLength <= 0 {
		return
	}

	as.stream.Write(sampleData[:sampleLength])

	as.cond.Signal()
}

func (as *audioStream) Capabilities() int {
	as.log.Info("audio stream capabilities requested")
	return 0
}

func (as *audioStream) Read(p []byte) (n int, err error) {
	as.cond.L.Lock()
	defer as.cond.L.Unlock()

	for as.stream.Len() == 0 && !as.closed {
		as.cond.Wait()
	}

	if as.closed && as.stream.Len() == 0 {
		return 0, io.EOF
	}

	return as.stream.Read(p)
}

func (as *audioStream) Close() error {
	as.Lock()
	as.closed = true
	as.stream.Reset()
	as.Unlock()

	as.cond.Broadcast()

	as.log.Info("audio stream closed", zap.String("action", "close"))
	return nil
}
