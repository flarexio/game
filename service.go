package surveillance

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

type Service interface {
	AcceptPeer(offer webrtc.SessionDescription, reply string) (*Peer, error)
}

func NewService(nc *nats.Conn) Service {
	return &service{
		log: zap.L().With(
			zap.String("service", "surveillance"),
		),
		nc: nc,
	}
}

type service struct {
	log *zap.Logger
	nc  *nats.Conn
	sync.Mutex
}

var configuration = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs: []string{
				"stun:stun.l.google.com:19302",
				"stun:stun1.l.google.com:19302",
				"stun:stun2.l.google.com:19302",
				"stun:stun3.l.google.com:19302",
				"stun:stun4.l.google.com:19302",
			},
		},
	},
}

func (svc *service) AcceptPeer(offer webrtc.SessionDescription, reply string) (*Peer, error) {
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
