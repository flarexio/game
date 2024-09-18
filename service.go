package surveillance

import (
	"encoding/json"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/pion/webrtc/v4"
)

type Service interface {
	AcceptPeer(offer webrtc.SessionDescription, reply string) (*Peer, error)
}

func NewService(nc *nats.Conn) Service {
	return &service{
		nc:    nc,
		peers: make([]*Peer, 0),
	}
}

type service struct {
	nc    *nats.Conn
	peers []*Peer
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

	sub, err := svc.nc.Subscribe(reply+".candidates.caller", func(msg *nats.Msg) {
		var candidate webrtc.ICECandidateInit
		if err := json.Unmarshal(msg.Data, &candidate); err != nil {
			return
		}

		conn.AddICECandidate(candidate)
	})

	if err != nil {
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

	peer := &Peer{conn, sub}

	svc.Lock()
	svc.peers = append(svc.peers, peer)
	svc.Unlock()

	return peer, nil
}

type Peer struct {
	*webrtc.PeerConnection
	sub *nats.Subscription
}
