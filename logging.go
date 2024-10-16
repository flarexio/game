package game

import (
	"github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

func LoggingMiddleware(log *zap.Logger) ServiceMiddleware {
	return func(next Service) Service {
		log := log.With(
			zap.String("service", "game"),
		)

		log.Info("service built")

		return &loggingMiddleware{log, next}
	}
}

type loggingMiddleware struct {
	log  *zap.Logger
	next Service
}

func (mw *loggingMiddleware) FindStream(name string) (*Stream, error) {
	log := mw.log.With(
		zap.String("action", "find_stream"),
		zap.String("stream", name),
	)

	stream, err := mw.next.FindStream(name)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	log.Info("stream got")

	return stream, nil
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

	log.Info("peer accepted")

	return peer, nil
}

func (mw *loggingMiddleware) Close() error {
	log := mw.log.With(
		zap.String("action", "close"),
	)

	err := mw.next.Close()
	if err != nil {
		log.Error(err.Error())
	}

	log.Info("service closed")

	return err
}
