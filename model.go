package game

import (
	"errors"
	"net/url"
	"strings"

	"github.com/pion/webrtc/v4"
	"gopkg.in/yaml.v3"

	"github.com/flarexio/game/nvstream"
)

type Config struct {
	Path    string    `yaml:"-"`
	WebRTC  WebRTC    `yaml:"webrtc"`
	Streams []*Stream `yaml:"streams"`
}

type WebRTC struct {
	ICEServers []*ICEServer `yaml:"iceServers"`
}

type ICEServer struct {
	Provider ICEProvider `yaml:"provider"`
	ID       string      `yaml:"id"`
	Token    string      `yaml:"token"`
}

type ICEProvider int

const (
	Google ICEProvider = iota
	Cloudflare
	Metered
)

func ParseICEProvider(provider string) (ICEProvider, error) {
	switch provider {
	case "google":
		return Google, nil
	case "cloudflare":
		return Cloudflare, nil
	case "metered":
		return Metered, nil
	default:
		return -1, errors.New("provider not supported")
	}
}

func (provider *ICEProvider) UnmarshalYAML(value *yaml.Node) error {
	var raw string
	if err := value.Decode(&raw); err != nil {
		return err
	}

	p, err := ParseICEProvider(raw)
	if err != nil {
		return err
	}

	*provider = p

	return nil
}

func (provider ICEProvider) String() string {
	switch provider {
	case Google:
		return "google"
	case Cloudflare:
		return "cloudflare"
	case Metered:
		return "metered"
	default:
		return "unknown"
	}
}

type Stream struct {
	Name      string
	Transport Transport
	Address   *url.URL
	NVStream  *nvstream.StreamConfiguration
	Video     *VideoTrack
	Audio     *AudioTrack
}

func (s *Stream) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Name      string                        `yaml:"name"`
		Transport Transport                     `yaml:"transport"`
		Address   string                        `yaml:"address"`
		NVStream  *nvstream.StreamConfiguration `yaml:"nvstream"`
		Video     *VideoTrack                   `yaml:"video"`
		Audio     *AudioTrack                   `yaml:"audio"`
	}

	if err := value.Decode(&raw); err != nil {
		return err
	}

	s.Name = raw.Name
	s.Transport = raw.Transport

	if raw.Address != "" {
		url, err := url.Parse(raw.Address)
		if err != nil {
			return err
		}

		s.Address = url
	}

	s.NVStream = raw.NVStream
	s.Video = raw.Video
	s.Audio = raw.Audio

	return nil
}

type Track interface {
	Address() *url.URL
	Codec() Codec
	Track() webrtc.TrackLocal
}

type VideoTrack struct {
	address *url.URL
	codec   Codec
	fps     float64
	track   webrtc.TrackLocal
}

func (video *VideoTrack) Address() *url.URL {
	return video.address
}

func (video *VideoTrack) Codec() Codec {
	return video.codec
}

func (video *VideoTrack) FPS() float64 {
	return video.fps
}

func (video *VideoTrack) Track() webrtc.TrackLocal {
	return video.track
}

func (video *VideoTrack) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Address string  `yaml:"address"`
		Codec   Codec   `yaml:"codec"`
		FPS     float64 `yaml:"fps"`
	}

	if err := value.Decode(&raw); err != nil {
		return err
	}

	if raw.Address != "" {
		url, err := url.Parse(raw.Address)
		if err != nil {
			return err
		}

		video.address = url
	}

	mimeType := raw.Codec.MimeType()
	if mimeType == "unknown" {
		return errors.New("codec unsupported")
	}

	if raw.Codec != CodecNone {
		if !strings.HasPrefix(mimeType, "video") {
			return errors.New("invalid codec")
		}
	}

	video.codec = raw.Codec
	video.fps = raw.FPS

	return nil
}

type AudioTrack struct {
	address *url.URL
	codec   Codec
	track   webrtc.TrackLocal
}

func (audio *AudioTrack) Address() *url.URL {
	return audio.address
}

func (audio *AudioTrack) Codec() Codec {
	return audio.codec
}

func (audio *AudioTrack) Track() webrtc.TrackLocal {
	return audio.track
}

func (audio *AudioTrack) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Address string
		Codec   Codec
	}

	if err := value.Decode(&raw); err != nil {
		return err
	}

	if raw.Address != "" {
		url, err := url.Parse(raw.Address)
		if err != nil {
			return err
		}

		audio.address = url
	}

	mimeType := raw.Codec.MimeType()
	if mimeType == "unknown" {
		return errors.New("codec unsupported")
	}

	if raw.Codec != CodecNone {
		if !strings.HasPrefix(mimeType, "audio") {
			return errors.New("invalid codec")
		}
	}

	audio.codec = raw.Codec

	return nil
}

type Transport string

const (
	TransportRaw  Transport = "raw"
	TransportRTP  Transport = "rtp"
	TransportRTSP Transport = "rtsp"
	TransportRTMP Transport = "rtmp"
	TransportHTTP Transport = "http"
	TransportNV   Transport = "nvstream"
)

type Codec string

const (
	CodecNone Codec = "none"
	CodecH264 Codec = "h264"
	CodecH265 Codec = "h265"
	CodecOpus Codec = "opus"
	CodecVP8  Codec = "vp8"
	CodecVP9  Codec = "vp9"
	CodecAV1  Codec = "av1"
	CodecG722 Codec = "g722"
	CodecPCMU Codec = "pcmu"
	CodecPCMA Codec = "pcma"
)

func (codec Codec) MimeType() string {
	switch codec {
	case CodecH264:
		return webrtc.MimeTypeH264
	case CodecH265:
		return webrtc.MimeTypeH265
	case CodecOpus:
		return webrtc.MimeTypeOpus
	case CodecVP8:
		return webrtc.MimeTypeVP8
	case CodecVP9:
		return webrtc.MimeTypeVP9
	case CodecAV1:
		return webrtc.MimeTypeAV1
	case CodecG722:
		return webrtc.MimeTypeG722
	case CodecPCMU:
		return webrtc.MimeTypePCMU
	case CodecPCMA:
		return webrtc.MimeTypePCMA
	default:
		return "unknown"
	}
}
