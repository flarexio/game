package surveillance

import (
	"github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

func LoggingMiddleware(log *zap.Logger) ServiceMiddleware {
	return func(next Service) Service {
		log := log.With(
			zap.String("service", "surveillance"),
		)

		log.Info("service built")

		return &loggingMiddleware{log, next}
	}
}

type loggingMiddleware struct {
	log  *zap.Logger
	next Service
}

func (mw *loggingMiddleware) ICEServers(provider ICEProvider) ([]webrtc.ICEServer, error) {
	log := mw.log.With(
		zap.String("action", "ice_servers"),
		zap.String("provider", provider.String()),
	)

	servers, err := mw.next.ICEServers(provider)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	log.Info("got servers", zap.Int("count", len(servers)))
	return servers, nil
}

func (mw *loggingMiddleware) AcceptPeer(offer webrtc.SessionDescription, reply string) (*Peer, error) {
	log := mw.log.With(
		zap.String("action", "accept_peer"),
		zap.String("reply", reply),
	)

	peer, err := mw.next.AcceptPeer(offer, reply)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	return peer, nil
}
