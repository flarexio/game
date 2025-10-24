package nvstream

import (
	"github.com/flarexio/game/thirdparty/moonlight"
)

type ContextKey string

const (
	CtxKeyStreamConfiguration ContextKey = "StreamConfiguration"
	CtxKeyRemoteInputAES      ContextKey = "RI"

	CtxKeyCSeq          ContextKey = "CSeq"
	CtxKeyClientVersion ContextKey = "ClientVersion"
	CtxKeySessionID     ContextKey = "SessionID"
)

func NewStreamConfiguration() *StreamConfiguration {
	return &StreamConfiguration{
		App:                      NvApp{Name: "Steam"},
		Width:                    1280,
		Height:                   720,
		RefreshRate:              60,
		LaunchRefreshRate:        60,
		Bitrate:                  10000,
		MaxPacketSize:            1024,
		Remote:                   moonlight.STREAM_CFG_AUTO,
		SOPS:                     true,
		EnableAdaptiveResolution: false,
		AudioConfiguration:       moonlight.AUDIO_CONFIGURATION_STEREO,
		SupportedVideoFormats:    moonlight.VIDEO_FORMAT_H264,
		AttachedGamepadMask:      0,
	}
}

type StreamConfiguration struct {
	App                           NvApp
	Width                         int
	Height                        int
	RefreshRate                   int
	LaunchRefreshRate             int
	ClientRefreshRateX100         int
	Bitrate                       int
	SOPS                          bool
	EnableAdaptiveResolution      bool
	PlayLocalAudio                bool
	MaxPacketSize                 int
	Remote                        int
	AudioConfiguration            moonlight.AudioConfiguration
	SupportedVideoFormats         int
	AttachedGamepadMask           int
	EncryptionFlags               int
	ColorRange                    int
	ColorSpace                    int
	PersistGamepadAfterDisconnect bool
}

func (cfg *StreamConfiguration) SetAttachedGamepadMaskByCount(count int) {
	cfg.AttachedGamepadMask = 0
	for i := 0; i < 4; i++ {
		if count > i {
			cfg.AttachedGamepadMask |= (1 << i)
		}
	}
}

type NvConnectionListener func()
