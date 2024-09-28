package surveillance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/nats-io/nats.go"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"github.com/pion/webrtc/v4/pkg/media/h264reader"
	"go.uber.org/zap"
)

type Service interface {
	// TODO: migrate to a dedicated ICE Server provider
	ICEServers(provider ICEProvider) ([]webrtc.ICEServer, error)
	AcceptPeer(offer webrtc.SessionDescription, reply string) (*Peer, error)
	Close() error
}

type ServiceMiddleware func(next Service) Service

func NewService(cfg *Config, nc *nats.Conn) Service {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	svc := &service{
		log: zap.L().With(
			zap.String("service", "surveillance"),
		),
		cfg:    cfg,
		nc:     nc,
		peers:  make([]*Peer, 0),
		cancel: cancel,
	}

	go svc.listenUDS(ctx)

	return svc
}

type service struct {
	log    *zap.Logger
	cfg    *Config
	nc     *nats.Conn
	peers  []*Peer
	track  *webrtc.TrackLocalStaticSample
	cancel context.CancelFunc
	sync.RWMutex
}

func (svc *service) listenUDS(ctx context.Context) {
	address := "/home/ar0660/video/input.sock"

	log := svc.log.With(
		zap.String("action", "listen_uds"),
		zap.String("address", address),
	)

	listener, err := net.Listen("unix", address)
	if err != nil {
		log.Error(err.Error())
		return
	}
	log.Info("unix socket opened")

	go func(ctx context.Context, listener net.Listener) {
		<-ctx.Done()

		listener.Close()
		log.Info("unix socket closed")
	}(ctx, listener)

	track, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{
		MimeType: webrtc.MimeTypeH264,
	}, "video", "pion")

	if err != nil {
		log.Error(err.Error())
		return
	}

	svc.track = track

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error(err.Error())
			return
		}

		go func(ctx context.Context, conn net.Conn) {
			log := log.With(
				zap.String("address", conn.RemoteAddr().String()),
			)

			reader, err := h264reader.NewReader(conn)
			if err != nil {
				log.Error(err.Error())
				return
			}

			log.Info("playing")

			for {
				select {
				case <-ctx.Done():
					log.Info("done")
					return

				default:
					nal, err := reader.NextNAL()
					if err != nil {
						log.Error(err.Error())
						return
					}

					track.WriteSample(media.Sample{
						Data:     nal.Data,
						Duration: 40 * time.Millisecond,
					})
				}
			}
		}(ctx, conn)
	}
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

	peer.Init()

	sub, err := svc.nc.Subscribe(reply+".candidates.caller", peer.candidateUpdatedHandler())
	if err != nil {
		return nil, err
	}

	peer.sub = sub

	if _, err := conn.AddTrack(svc.track); err != nil {
		return nil, err
	}

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

	svc.Lock()
	svc.peers = append(svc.peers, peer)
	svc.Unlock()

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

func (peer *Peer) ICEConnectionStateChangeHandler(cancel context.CancelFunc) func(webrtc.ICEConnectionState) {
	log := peer.log.With(
		zap.String("handler", "ice_connection_state_change"),
	)

	return func(state webrtc.ICEConnectionState) {
		log.Info("connection state has changed",
			zap.String("state", state.String()))

		if state == webrtc.ICEConnectionStateConnected {
			cancel()
		}
	}
}

func (svc *service) Close() error {
	if svc.cancel != nil {
		svc.cancel()
		svc.cancel = nil
	}

	return nil
}
