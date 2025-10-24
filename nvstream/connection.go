package nvstream

import (
	"context"
	"errors"

	"github.com/flarexio/game/thirdparty/moonlight"
)

type NvConnection interface {
	StartApp(ctx context.Context, app NvApp) error
	StopApp(ctx context.Context) error
}

func NewConnection(host string, uniqueID string, stream *StreamConfiguration) (NvConnection, error) {
	ri, err := moonlight.NewRemoteInputAES()
	if err != nil {
		return nil, err
	}

	return &nvConnection{
		uniqueID: uniqueID,
		host:     host,
		stream:   stream,
		ri:       ri,
	}, nil
}

type nvConnection struct {
	uniqueID string
	host     string
	stream   *StreamConfiguration
	ri       *moonlight.RemoteInputAES
}

func (conn *nvConnection) StartApp(ctx context.Context, app NvApp) error {
	http, err := NewHTTP(conn.uniqueID, conn.host)
	if err != nil {
		return err
	}

	info, err := http.ServerInfo()
	if err != nil {
		return err
	}

	if info.PairStatus != int(PairStatePaired) {
		return errors.New("device not paired with computer")
	}

	isNvidiaServerSoftware := false

	negotiatedHDR := (conn.stream.SupportedVideoFormats & moonlight.VIDEO_FORMAT_MASK_10BIT) != 0
	if (info.ServerCodecModeSupport&0x20200) == 0 && negotiatedHDR {
		return errors.New("server does not support HDR streaming")
	}

	if conn.stream.Width > 4096 || conn.stream.Height > 4096 {
		if (info.ServerCodecModeSupport&0x200 == 0) && isNvidiaServerSoftware {
			return errors.New("server does not support resolutions above 4K pixels")
		}

		if (conn.stream.SupportedVideoFormats & ^moonlight.VIDEO_FORMAT_MASK_H264) == 0 {
			return errors.New("server does not support resolutions above 4K pixels for H264 streams")
		}
	}

	if conn.stream.Width > 2160 && !info.Supports4K() {
		return errors.New("server does not support resolutions above 4K pixels")
	}

	var (
		negotiatedRemoteStreaming = conn.stream.Remote
		negotiatedPacketSize      = conn.stream.MaxPacketSize
	)

	if conn.stream.Remote == moonlight.STREAM_CFG_REMOTE {
		negotiatedPacketSize = 1024
	}

	ctx = context.WithValue(ctx, CtxKeyStreamConfiguration, conn.stream)
	ctx = context.WithValue(ctx, CtxKeyRemoteInputAES, conn.ri)

	rtspSessionURL, err := http.LaunchApp(ctx, app.ID, false)
	if err != nil {
		return err
	}

	serverInfo := moonlight.ServerInformation{
		Address:                info.Hostname,
		AppVersion:             info.AppVersion,
		GfeVersion:             info.GfeVersion,
		ServerCodecModeSupport: info.ServerCodecModeSupport,
		RTSPSessionURL:         rtspSessionURL,
	}

	streamConfig := moonlight.StreamConfiguration{
		Width:                 conn.stream.Width,
		Height:                conn.stream.Height,
		FPS:                   conn.stream.RefreshRate,
		Bitrate:               conn.stream.Bitrate,
		PacketSize:            negotiatedPacketSize,
		StreamingRemotely:     negotiatedRemoteStreaming,
		AudioConfiguration:    conn.stream.AudioConfiguration,
		SupportedVideoFormats: conn.stream.SupportedVideoFormats,
		ClientRefreshRateX100: conn.stream.ClientRefreshRateX100,
		EncryptionFlags:       moonlight.ENCFLG_ALL,
		ColorSpace:            conn.stream.ColorSpace,
		ColorRange:            conn.stream.ColorRange,
		RemoteInputAES:        conn.ri,
	}

	return moonlight.StartConnection(serverInfo, streamConfig)
}

func (conn *nvConnection) StopApp(ctx context.Context) error {
	moonlight.StopConnection()

	http, err := NewHTTP(conn.uniqueID, conn.host)
	if err != nil {
		return err
	}

	return http.QuitApp(ctx)
}
