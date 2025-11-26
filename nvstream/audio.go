package nvstream

import (
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
		log:     log,
		packets: make(chan []byte, 500),
		closed:  false,
	}
}

type audioStream struct {
	log *zap.Logger

	sampleDuration time.Duration
	packets        chan []byte
	closed         bool
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
	as.log.Info("audio stream cleaned up", zap.String("action", "cleanup"))
}

func (as *audioStream) AudioRendererDecodeAndPlaySample(sampleData []byte, sampleLength int) {
	as.Lock()
	closed := as.closed
	as.Unlock()

	if closed || sampleLength == 0 {
		return
	}

	data := make([]byte, sampleLength)
	copy(data, sampleData[:sampleLength])

	select {
	case as.packets <- data:
	default:
		as.log.Warn("audio packet dropped due to full buffer")
		as.flushQueue()
		as.packets <- data
	}
}

func (as *audioStream) flushQueue() {
	for {
		select {
		case <-as.packets:
		default:
			return
		}
	}
}

func (as *audioStream) Capabilities() int {
	as.log.Info("audio stream capabilities requested")
	return 0
}

func (as *audioStream) Read(p []byte) (n int, err error) {
	as.Lock()
	closed := as.closed
	as.Unlock()

	if closed {
		return 0, io.EOF
	}

	packet, ok := <-as.packets
	if !ok {
		return 0, io.EOF
	}

	n = copy(p, packet)
	return n, nil
}

func (as *audioStream) Close() error {
	as.Lock()
	if !as.closed {
		as.closed = true
		close(as.packets)
	}
	as.Unlock()

	as.log.Info("audio stream closed", zap.String("action", "close"))
	return nil
}
