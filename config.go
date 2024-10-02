package game

import (
	"errors"
	"net/url"
	"strings"

	"github.com/pion/webrtc/v4"
	"gopkg.in/yaml.v3"
)

type Config struct {
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
	Name    string
	Origins []*Origin
}

func (stream *Stream) VideoTrack(index ...int) (webrtc.TrackLocal, error) {
	i := 0
	if len(index) > 0 {
		i = index[0]
	}

	if i >= len(stream.Origins) {
		return nil, errors.New("track not found")
	}

	videoTrack := stream.Origins[i].Video

	if videoTrack == nil {
		return nil, errors.New("track not found")
	}

	return videoTrack.track, nil
}

func (stream *Stream) AudioTrack(index ...int) (webrtc.TrackLocal, error) {
	i := 0
	if len(index) > 0 {
		i = index[0]
	}

	if i >= len(stream.Origins) {
		return nil, errors.New("track not found")
	}

	audioTrack := stream.Origins[i].Audio

	if audioTrack == nil {
		return nil, errors.New("track not found")
	}

	return audioTrack.track, nil
}

type Origin struct {
	Transport Transport
	Video     *VideoTrack
	Audio     *AudioTrack
}

type Track interface {
	Address() *url.URL
	Codec() Codec
	Track() webrtc.TrackLocal
}

type VideoTrack struct {
	address *url.URL
	codec   Codec
	track   webrtc.TrackLocal
}

func (video *VideoTrack) Address() *url.URL {
	return video.address
}

func (video *VideoTrack) Codec() Codec {
	return video.codec
}

func (video *VideoTrack) Track() webrtc.TrackLocal {
	return video.track
}

func (video *VideoTrack) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Address string
		Codec   Codec
	}

	if err := value.Decode(&raw); err != nil {
		return err
	}

	address, err := url.Parse(raw.Address)
	if err != nil {
		return err
	}

	video.address = address

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

func (track *AudioTrack) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Address string
		Codec   Codec
	}

	if err := value.Decode(&raw); err != nil {
		return err
	}

	address, err := url.Parse(raw.Address)
	if err != nil {
		return err
	}

	track.address = address

	mimeType := raw.Codec.MimeType()
	if mimeType == "unknown" {
		return errors.New("codec unsupported")
	}

	if raw.Codec != CodecNone {
		if !strings.HasPrefix(mimeType, "audio") {
			return errors.New("invalid codec")
		}
	}

	track.codec = raw.Codec

	return nil
}

type Transport string

const (
	TransportRaw  Transport = "raw"
	TransportRTP  Transport = "rtp"
	TransportRTSP Transport = "rtsp"
	TransportRTMP Transport = "rtmp"
	TransportHTTP Transport = "http"
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
