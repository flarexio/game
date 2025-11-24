package game

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	"github.com/pion/webrtc/v4/pkg/media/oggreader"
	"go.uber.org/zap"

	"github.com/flarexio/core/model"
	"github.com/flarexio/game/nvstream"
	"github.com/flarexio/game/thirdparty/moonlight"
)

type Service interface {
	FindStream(name string) (*Stream, error)

	// TODO: migrate to a dedicated ICE Server provider
	ICEServers(provider ICEProvider) ([]webrtc.ICEServer, error)
	AcceptPeer(offer webrtc.SessionDescription, reply string) (*Peer, error)
	Close() error
}

type ServiceMiddleware func(next Service) Service

func NewService(cfg *Config, nc *nats.Conn) (Service, error) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	svc := &service{
		log: zap.L().With(
			zap.String("service", "game"),
		),
		cfg:    cfg,
		nc:     nc,
		peers:  make([]*Peer, 0),
		cancel: cancel,
	}

	err := svc.buildStreams(ctx, cfg.Streams)
	if err != nil {
		return nil, err
	}

	gamepad, err := NewGamepad()
	if err != nil {
		return nil, err
	}

	if err := gamepad.Connect(); err != nil {
		return nil, err
	}

	svc.gamepad = gamepad

	return svc, nil
}

type service struct {
	log     *zap.Logger
	cfg     *Config
	nc      *nats.Conn
	streams map[string]*Stream
	peers   []*Peer
	gamepad Gamepad
	cancel  context.CancelFunc
	sync.RWMutex
}

func (svc *service) buildStreams(ctx context.Context, streams []*Stream) error {
	streamMap := make(map[string]*Stream)
	for _, stream := range streams {
		switch stream.Transport {
		case TransportRaw:
			if video := stream.Video; video != nil {
				if video.Codec() == CodecNone {
					return errors.New("video codec not specified")
				}

				trackID := stream.Name + "_video"

				track, err := webrtc.NewTrackLocalStaticSample(
					webrtc.RTPCodecCapability{
						MimeType: video.Codec().MimeType(),
					}, trackID, stream.Name,
				)

				if err != nil {
					return err
				}

				video.track = track

				go svc.listen(ctx, video)
			}

			if audio := stream.Audio; audio != nil {
				if audio.Codec() == CodecNone {
					return errors.New("audio codec not specified")
				}

				trackID := stream.Name + "_audio"

				track, err := webrtc.NewTrackLocalStaticSample(
					webrtc.RTPCodecCapability{
						MimeType: audio.Codec().MimeType(),
					}, trackID, stream.Name,
				)

				if err != nil {
					return err
				}

				audio.track = track

				go svc.listen(ctx, audio)
			}

		case TransportNV:
			// Resolve NVStream App
			host := stream.Address.Hostname()

			http, err := nvstream.NewHTTP("MyGameClient", host)
			if err != nil {
				return err
			}

			appList, err := http.AppList()
			if err != nil {
				return err
			}

			var app nvstream.NvApp
			for _, a := range appList {
				if !strings.Contains(a.Name, stream.NVStream.App.Name) {
					continue
				}

				app = a
			}

			if (app == nvstream.NvApp{}) {
				return errors.New("nvstream app not found: " + stream.NVStream.App.Name)
			}

			stream.NVStream.App = app

			conn, err := nvstream.NewConnection(host, "MyGameClient", stream.NVStream)
			if err != nil {
				return err
			}

			vs := nvstream.NewVideoStream()
			as := nvstream.NewAudioStream()

			moonlight.SetupCallbacks(conn, vs, as)

			if err := conn.StartApp(ctx, app); err != nil {
				return err
			}

			if video := stream.Video; video != nil {
				trackID := stream.Name + "_video"
				// trackID := "video_0"

				track, err := webrtc.NewTrackLocalStaticSample(
					webrtc.RTPCodecCapability{
						MimeType: video.Codec().MimeType(),
					}, trackID, stream.Name,
				)

				if err != nil {
					return err
				}

				video.track = track

				if err := svc.trackHandler(ctx, vs, video); err != nil {
					return err
				}
			}

			if audio := stream.Audio; audio != nil {
				// TODO: support more audio codecs
			}

		default:
			return errors.New("transport unsupported")
		}

		streamMap[stream.Name] = stream
	}

	svc.streams = streamMap

	return nil
}

func (svc *service) listen(ctx context.Context, track Track) {
	url := track.Address()

	network := url.Scheme

	address := url.Host
	if url.Scheme == "unix" {
		address = url.Path
	}

	log := svc.log.With(
		zap.String("action", "listen"),
		zap.String("network", network),
		zap.String("address", address),
	)

	if strings.HasPrefix(network, "udp") {
		addr, err := net.ResolveUDPAddr(network, address)
		if err != nil {
			log.Error(err.Error())
			return
		}

		conn, err := net.ListenUDP(network, addr)
		if err != nil {
			log.Error(err.Error())
			return
		}

		log.Info("socket opened")

		ctx = context.WithValue(ctx, model.Logger, log)

		if err := svc.trackHandler(ctx, conn, track); err != nil {
			log.Error(err.Error())
		}

		return
	}

	listener, err := net.Listen(network, address)
	if err != nil {
		log.Error(err.Error())
		return
	}

	log.Info("socket opened")

	go func(ctx context.Context, listener net.Listener) {
		<-ctx.Done()

		listener.Close()
		log.Info("socket closed")
	}(ctx, listener)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Error(err.Error())
			return
		}

		log := log.With(
			zap.String("remote", conn.RemoteAddr().String()),
		)

		ctx = context.WithValue(ctx, model.Logger, log)

		if err := svc.trackHandler(ctx, conn, track); err != nil {
			log.Error(err.Error())
		}
	}
}

func (svc *service) trackHandler(ctx context.Context, r io.ReadCloser, track Track) error {
	switch track := track.(type) {
	case *VideoTrack:
		switch track.Codec() {
		case CodecH264:
			go svc.h264Handler(ctx, r, track)

		default:
			return errors.New("video codec unsupported")
		}

	case *AudioTrack:
		switch track.Codec() {
		case CodecOpus:
			go svc.oggHandler(ctx, r, track)

		default:
			return errors.New("audio codec unsupported")
		}

	default:
		return errors.New("track type unsupported")
	}

	return nil
}

func (svc *service) h264Handler(ctx context.Context, r io.ReadCloser, video *VideoTrack) {
	log, ok := ctx.Value(model.Logger).(*zap.Logger)
	if !ok {
		log = svc.log
	}

	log = log.With(
		zap.String("track", "video"),
		zap.String("container", "raw"),
		zap.String("codec", string(video.Codec())),
		zap.Float64("fps", video.FPS()),
	)

	frameDuration := time.Second / time.Duration(video.FPS())

	track, ok := video.Track().(*webrtc.TrackLocalStaticSample)
	if !ok {
		log.Error("invalid type")
		return
	}

	reader, err := h264reader.NewReader(r)
	if err != nil {
		log.Error(err.Error())
		return
	}

	log.Info("playing")

	for {
		select {
		case <-ctx.Done():
			r.Close()
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
				Duration: frameDuration,
			})
		}
	}
}

func (svc *service) oggHandler(ctx context.Context, r io.ReadCloser, audio *AudioTrack) {
	log, ok := ctx.Value(model.Logger).(*zap.Logger)
	if !ok {
		log = svc.log
	}

	log = log.With(
		zap.String("track", "audio"),
		zap.String("container", "ogg"),
		zap.String("codec", string(audio.Codec())),
	)

	track, ok := audio.Track().(*webrtc.TrackLocalStaticSample)
	if !ok {
		log.Error("invalid type")
		return
	}

	reader, _, err := oggreader.NewWith(r)
	if err != nil {
		log.Error(err.Error())
		return
	}

	log.Info("playing")

	var lastGranule uint64
	for {
		select {
		case <-ctx.Done():
			r.Close()
			log.Info("done")
			return

		default:
			payload, header, err := reader.ParseNextPage()
			if err != nil {
				log.Error(err.Error())
				return
			}

			sampleCount := float64(header.GranulePosition - lastGranule)
			lastGranule = header.GranulePosition
			sampleDuration := time.Duration((sampleCount/48000)*1000) * time.Millisecond

			track.WriteSample(media.Sample{
				Data:     payload,
				Duration: sampleDuration,
			})
		}
	}
}

func (svc *service) FindStream(name string) (*Stream, error) {
	stream, ok := svc.streams[name]
	if !ok {
		return nil, errors.New("stream not found")
	}

	return stream, nil
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
		gamepad: svc.gamepad,
	}

	peer.Init()

	sub, err := svc.nc.Subscribe(reply+".candidates.caller", peer.candidateUpdatedHandler())
	if err != nil {
		return nil, err
	}

	peer.sub = sub

	stream, err := svc.FindStream("gamestream")
	if err != nil {
		return nil, err
	}

	videoTrack := stream.Video.Track()
	if videoTrack == nil {
		return nil, errors.New("video track not found")
	}

	if _, err := conn.AddTrack(videoTrack); err != nil {
		return nil, err
	}

	// audioTrack := stream.Audio.Track()
	// if audioTrack == nil {
	// 	return nil, errors.New("audio track not found")
	// }

	// if _, err := conn.AddTrack(audioTrack); err != nil {
	// 	return nil, err
	// }

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
	log     *zap.Logger
	sub     *nats.Subscription
	gamepad Gamepad
}

func (peer *Peer) Init() {
	log := peer.log

	peer.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		log.Info("connection state updated",
			zap.String("state", state.String()))

		if state == webrtc.PeerConnectionStateConnected {
			moonlight.RequestIDRFrame()
		}
	})

	peer.OnDataChannel(func(dc *webrtc.DataChannel) {
		dc.OnMessage(func(msg webrtc.DataChannelMessage) {
			switch dc.Label() {
			case "gamepad":
				report := NewXBoxGamepadReport(
					binary.BigEndian.Uint16(msg.Data[0:2]),
					msg.Data[2],
					msg.Data[3],
					int16(binary.BigEndian.Uint16(msg.Data[4:6])),
					int16(binary.BigEndian.Uint16(msg.Data[6:8])),
					int16(binary.BigEndian.Uint16(msg.Data[8:10])),
					int16(binary.BigEndian.Uint16(msg.Data[10:12])),
				)

				err := peer.gamepad.Update(report)
				if err != nil {
					log.Error(err.Error(),
						zap.String("label", "gamepad"))
				}
			}
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
	if svc.gamepad != nil {
		svc.gamepad.Close()
		svc.gamepad = nil
	}

	if svc.cancel != nil {
		svc.cancel()
		svc.cancel = nil
	}

	return nil
}
