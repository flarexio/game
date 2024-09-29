package game

import (
	"errors"

	"gopkg.in/yaml.v3"
)

type Config struct {
	WebRTC WebRTCConfig `yaml:"webrtc"`
}

type WebRTCConfig struct {
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
