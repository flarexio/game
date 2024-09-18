package surveillance

import (
	"encoding/json"
	"strings"

	"github.com/nats-io/nats.go/micro"
	"github.com/pion/webrtc/v4"
)

func AcceptPeerHandler(svc Service) micro.HandlerFunc {
	return func(r micro.Request) {
		var offer *webrtc.SessionDescription
		if err := json.Unmarshal(r.Data(), &offer); err != nil {
			r.Error("400", err.Error(), nil)
			return
		}

		reply, ok := strings.CutPrefix(r.Reply(), ".sdp.answer")
		if !ok {
			r.Error("400", "invalid reply", nil)
			return
		}

		peer, err := svc.AcceptPeer(*offer, reply)
		if err != nil {
			r.Error("417", err.Error(), nil)
			return
		}

		answer := peer.LocalDescription()
		r.RespondJSON(&answer)
	}
}
