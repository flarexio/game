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
		log:         log,
		frameChan:   make(chan audioFrame, 128),
		maxBufferMs: 200,
		closed:      false,
	}
}

type audioFrame struct {
	data      []byte
	timestamp time.Time
}

type audioStream struct {
	log *zap.Logger

	sampleDuration time.Duration
	frameChan      chan audioFrame
	maxBufferMs    int
	closed         bool
	closeOnce      sync.Once
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

	as.sampleDuration = time.Duration(durationMs) * time.Millisecond

	log.Info("audio stream initialized successfully",
		zap.String("layout", layout),
		zap.Int("sample_rate", sampleRate),
		zap.Duration("duration", as.sampleDuration),
	)

	return 0
}

func (as *audioStream) SampleDuration() time.Duration {
	return as.sampleDuration
}

func (as *audioStream) Start() {
	as.log.Info("audio stream started", zap.String("action", "start"))
}

func (as *audioStream) Stop() {
	as.log.Info("audio stream stopped", zap.String("action", "stop"))
}

func (as *audioStream) Cleanup() {
	as.closeOnce.Do(func() {
		as.closed = true
		close(as.frameChan)
	})

	as.log.Info("audio stream cleaned up", zap.String("action", "cleanup"))
}

func (as *audioStream) PlayEncodedSample(sampleData []byte, sampleLength int) {
	if as.closed || sampleLength == 0 {
		return
	}

	frame := audioFrame{
		data:      append([]byte{}, sampleData[:sampleLength]...),
		timestamp: time.Now(),
	}

	select {
	case as.frameChan <- frame:
		// ok
	default:
		// channel full, drop oldest frame
		<-as.frameChan
		as.frameChan <- frame
		as.log.Warn("audio stream buffer full, dropping oldest frame", zap.String("action", "play_encoded_sample"))
	}
}

func (as *audioStream) Capabilities() int {
	as.log.Info("audio stream capabilities requested")
	return 0
}

func (as *audioStream) Read(p []byte) (n int, err error) {
	for {
		if as.closed {
			return 0, io.EOF
		}

		frame, ok := <-as.frameChan
		if !ok {
			return 0, io.EOF
		}

		if time.Since(frame.timestamp) > time.Duration(as.maxBufferMs)*time.Millisecond {
			continue
		}

		n = copy(p, frame.data)
		return n, nil
	}
}

func (as *audioStream) Close() error {
	as.Cleanup()
	as.log.Info("audio stream closed", zap.String("action", "close"))
	return nil
}
