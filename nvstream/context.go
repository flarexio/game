package nvstream

import (
	"gopkg.in/yaml.v3"

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

func DefaultStreamConfiguration() *StreamConfiguration {
	return &StreamConfiguration{
		App:                           NvApp{Name: "Steam"},
		Width:                         1920,
		Height:                        1080,
		RefreshRate:                   60,
		LaunchRefreshRate:             60,
		ClientRefreshRateX100:         6000,
		Bitrate:                       10000,
		SOPS:                          true,
		EnableAdaptiveResolution:      false,
		PlayLocalAudio:                false,
		MaxPacketSize:                 1024,
		Remote:                        moonlight.STREAM_CFG_AUTO,
		AudioConfiguration:            moonlight.AUDIO_CONFIGURATION_STEREO,
		SupportedVideoFormats:         []moonlight.VideoFormat{moonlight.VIDEO_FORMAT_H264},
		AttachedGamepadMask:           0,
		EncryptionFlags:               moonlight.ENCFLG_NONE,
		ColorRange:                    moonlight.COLOR_RANGE_LIMITED,
		ColorSpace:                    moonlight.COLORSPACE_REC_709,
		PersistGamepadAfterDisconnect: false,
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
	Remote                        moonlight.StreamingRemotely
	AudioConfiguration            moonlight.AudioConfiguration
	SupportedVideoFormats         []moonlight.VideoFormat
	AttachedGamepadMask           int
	EncryptionFlags               moonlight.EncryptionFlags
	ColorRange                    moonlight.ColorRange
	ColorSpace                    moonlight.ColorSpace
	PersistGamepadAfterDisconnect bool
}

func (cfg *StreamConfiguration) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		App                           string   `yaml:"app"`
		Width                         int      `yaml:"width"`
		Height                        int      `yaml:"height"`
		RefreshRate                   int      `yaml:"refreshRate"`
		LaunchRefreshRate             int      `yaml:"launchRefreshRate"`
		ClientRefreshRateX100         int      `yaml:"clientRefreshRateX100"`
		Bitrate                       int      `yaml:"bitrate"`
		SOPS                          bool     `yaml:"sops"`
		EnableAdaptiveResolution      bool     `yaml:"enableAdaptiveResolution"`
		PlayLocalAudio                bool     `yaml:"playLocalAudio"`
		MaxPacketSize                 int      `yaml:"maxPacketSize"`
		Remote                        string   `yaml:"remote"`
		AudioConfiguration            string   `yaml:"audioConfiguration"`
		SupportedVideoFormats         []string `yaml:"supportedVideoFormats"`
		AttachedGamepadMask           int      `yaml:"attachedGamepadMask"`
		EncryptionFlags               string   `yaml:"encryptionFlags"`
		ColorRange                    string   `yaml:"colorRange"`
		ColorSpace                    string   `yaml:"colorSpace"`
		PersistGamepadAfterDisconnect bool     `yaml:"persistGamepadAfterDisconnect"`
	}

	if err := value.Decode(&raw); err != nil {
		return err
	}

	cfg.App = NvApp{Name: raw.App}
	cfg.Width = raw.Width
	cfg.Height = raw.Height
	cfg.RefreshRate = raw.RefreshRate
	cfg.LaunchRefreshRate = raw.LaunchRefreshRate
	cfg.ClientRefreshRateX100 = raw.ClientRefreshRateX100
	cfg.Bitrate = raw.Bitrate
	cfg.SOPS = raw.SOPS
	cfg.EnableAdaptiveResolution = raw.EnableAdaptiveResolution
	cfg.PlayLocalAudio = raw.PlayLocalAudio
	cfg.MaxPacketSize = raw.MaxPacketSize

	remote, err := moonlight.ParseStreamingRemotely(raw.Remote)
	if err != nil {
		return err
	}
	cfg.Remote = remote

	audioConfig, err := moonlight.ParseAudioConfiguration(raw.AudioConfiguration)
	if err != nil {
		return err
	}
	cfg.AudioConfiguration = audioConfig

	supportedVideoFormats := make([]moonlight.VideoFormat, len(raw.SupportedVideoFormats))
	for i, v := range raw.SupportedVideoFormats {
		format, err := moonlight.ParseVideoFormat(v)
		if err != nil {
			return err
		}
		supportedVideoFormats[i] = format
	}
	cfg.SupportedVideoFormats = supportedVideoFormats

	cfg.AttachedGamepadMask = raw.AttachedGamepadMask

	encryptionFlags, err := moonlight.ParseEncryptionFlags(raw.EncryptionFlags)
	if err != nil {
		return err
	}
	cfg.EncryptionFlags = encryptionFlags

	colorRange, err := moonlight.ParseColorRange(raw.ColorRange)
	if err != nil {
		return err
	}
	cfg.ColorRange = colorRange

	colorSpace, err := moonlight.ParseColorSpace(raw.ColorSpace)
	if err != nil {
		return err
	}
	cfg.ColorSpace = colorSpace

	cfg.PersistGamepadAfterDisconnect = raw.PersistGamepadAfterDisconnect

	return nil
}

func (cfg *StreamConfiguration) SetAttachedGamepadMaskByCount(count int) {
	cfg.AttachedGamepadMask = 0
	for i := 0; i < 4; i++ {
		if count > i {
			cfg.AttachedGamepadMask |= (1 << i)
		}
	}
}

func (cfg *StreamConfiguration) SupportedVideoFormatsBitmask() moonlight.VideoFormatMask {
	var bitmask int
	for _, format := range cfg.SupportedVideoFormats {
		bitmask |= int(format)
	}

	return moonlight.VideoFormatMask(bitmask)
}
