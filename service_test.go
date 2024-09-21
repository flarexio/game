package surveillance

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestICEServers(t *testing.T) {
	assert := assert.New(t)

	cfg := &Config{
		WebRTC: WebRTCConfig{
			ICEServers: []*ICEServer{
				{
					Provider: Google,
				},
			},
		},
	}

	svc := NewService(cfg, nil)
	for _, cfg := range cfg.WebRTC.ICEServers {
		switch cfg.Provider {
		case Google:
			servers, err := svc.ICEServers(Google)
			if err != nil {
				assert.Fail(err.Error())
				return
			}

			assert.Len(servers, 1)
			assert.Len(servers[0].URLs, 5)

		case Cloudflare:
			servers, err := svc.ICEServers(Cloudflare)
			if err != nil {
				assert.Fail(err.Error())
				return
			}

			assert.Len(servers, 1)
			assert.Len(servers[0].URLs, 4)

		case Metered:
			servers, err := svc.ICEServers(Metered)
			if err != nil {
				assert.Fail(err.Error())
				return
			}

			assert.Len(servers, 5)
		}
	}
}
