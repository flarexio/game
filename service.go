package surveillance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/nats-io/nats.go"
	"github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

type Service interface {
	// TODO: migrate to a dedicated ICE Server provider
	ICEServers(provider ICEProvider) ([]webrtc.ICEServer, error)
	AcceptPeer(offer webrtc.SessionDescription, reply string) (*Peer, error)
}

type ServiceMiddleware func(next Service) Service

func NewService(cfg *Config, nc *nats.Conn) Service {
	return &service{
		log: zap.L().With(
			zap.String("service", "surveillance"),
		),
		cfg: cfg,
		nc:  nc,
	}
}

type service struct {
	log *zap.Logger
	cfg *Config
	nc  *nats.Conn
	sync.Mutex
}

func (svc *service) ICEServers(provider ICEProvider) ([]webrtc.ICEServer, error) {
	var cfg *ICEServer
	for _, server := range svc.cfg.WebRTC.ICEServers {
		if server.Provider == provider {
			cfg = server
			break
		}
	}

	if cfg == nil {
		err := errors.New("provider not supported")
		return nil, err
	}

	switch cfg.Provider {
	case Google:
		return []webrtc.ICEServer{
			{
				URLs: []string{
					"stun:stun.l.google.com:19302",
					"stun:stun1.l.google.com:19302",
					"stun:stun2.l.google.com:19302",
					"stun:stun3.l.google.com:19302",
					"stun:stun4.l.google.com:19302",
				},
			},
		}, nil

	case Cloudflare:
		client := resty.New().
			SetBaseURL("https://rtc.live.cloudflare.com/v1")

		path := fmt.Sprintf("/turn/keys/%s/credentials/generate", cfg.ID)

		var config struct {
			ICEServers webrtc.ICEServer `json:"iceServers"`
		}

		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetAuthToken(cfg.Token).
			SetBody(`{ "ttl": 86400 }`).
			SetResult(&config).
			Post(path)

		if err != nil {
			return nil, err
		}

		if resp.StatusCode() != http.StatusCreated {
			var errMsg struct {
				Error string `json:"error"`
			}

			err := json.Unmarshal(resp.Body(), &errMsg)
			if err != nil {
				return nil, err
			}

			return nil, errors.New(errMsg.Error)
		}

		return []webrtc.ICEServer{config.ICEServers}, nil

	case Metered:
		baseURL := fmt.Sprintf("https://%s.metered.live/api/v1", cfg.ID)

		client := resty.New().
			SetBaseURL(baseURL)

		type ICEServer struct {
			URLs       string `json:"urls"`
			Username   string `json:"username"`
			Credential string `json:"credential"`
		}

		var raws []ICEServer
		resp, err := client.R().
			SetQueryParam("apiKey", cfg.Token).
			SetResult(&raws).
			Get("/turn/credentials")

		if err != nil {
			return nil, err
		}

		if resp.StatusCode() != http.StatusOK {
			var errMsg struct {
				Error string `json:"error"`
			}

			err := json.Unmarshal(resp.Body(), &errMsg)
			if err != nil {
				return nil, err
			}

			return nil, errors.New(errMsg.Error)
		}

		servers := make([]webrtc.ICEServer, len(raws))
		for i, raw := range raws {
			servers[i] = webrtc.ICEServer{
				URLs:       []string{raw.URLs},
				Username:   raw.Username,
				Credential: raw.Credential,
			}
		}

		return servers, nil

	default:
		return nil, errors.New("provider not supported")
	}
}

func (svc *service) AcceptPeer(offer webrtc.SessionDescription, reply string) (*Peer, error) {
	servers, err := svc.ICEServers(Google)
	if err != nil {
		return nil, err
	}

	configuration := webrtc.Configuration{
		ICEServers: servers,
	}

	conn, err := webrtc.NewPeerConnection(configuration)
	if err != nil {
		return nil, err
	}

	conn.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		bs, err := json.Marshal(&candidate)
		if err != nil {
			return
		}

		svc.nc.Publish(reply+".candidates.callee", bs)
	})

	inbox := strings.TrimPrefix(reply, "peers.negotiation.")

	peer := &Peer{
		PeerConnection: conn,
		log: svc.log.With(
			zap.String("peer", inbox),
		),
	}

	sub, err := svc.nc.Subscribe(reply+".candidates.caller", peer.candidateUpdatedHandler())
	if err != nil {
		return nil, err
	}

	peer.sub = sub

	peer.Init()

	if err := conn.SetRemoteDescription(offer); err != nil {
		return nil, err
	}

	answer, err := conn.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}

	gatherComplete := webrtc.GatheringCompletePromise(conn)

	if err := conn.SetLocalDescription(answer); err != nil {
		return nil, err
	}

	<-gatherComplete

	return peer, nil
}

type Peer struct {
	*webrtc.PeerConnection
	log *zap.Logger
	sub *nats.Subscription
}

func (peer *Peer) Init() {
	log := peer.log

	peer.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Info("connection state updated",
			zap.String("state", state.String()))
	})

	peer.OnDataChannel(func(dc *webrtc.DataChannel) {
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			log.Info("message arrived",
				zap.String("label", dc.Label()),
				zap.String("message", string(msg.Data)),
			)
		})
	})
}

func (peer *Peer) candidateUpdatedHandler() nats.MsgHandler {
	log := peer.log.With(
		zap.String("handler", "candidate_updated"),
	)

	return func(msg *nats.Msg) {
		var candidate webrtc.ICECandidateInit
		if err := json.Unmarshal(msg.Data, &candidate); err != nil {
			log.Error(err.Error())
			return
		}

		if err := peer.AddICECandidate(candidate); err != nil {
			log.Error(err.Error())
			return
		}

		log.Info("candidate added",
			zap.String("candidate", candidate.Candidate))
	}
}
