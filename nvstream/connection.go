package nvstream

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/flarexio/game/thirdparty/moonlight"
)

type NvConnection interface {
	StartApp(ctx context.Context, app NvApp) error
	StopApp(ctx context.Context) error
	moonlight.ConnectionListener
}

func NewConnection(host string, uniqueID string, stream *StreamConfiguration) (NvConnection, error) {
	log := zap.L().With(
		zap.String("component", "nvstream.connection"),
		zap.String("host", host),
		zap.String("unique_id", uniqueID),
	)

	ri, err := moonlight.NewRemoteInputAES()
	if err != nil {
		return nil, err
	}

	return &nvConnection{
		log:      log,
		uniqueID: uniqueID,
		host:     host,
		stream:   stream,
		ri:       ri,
	}, nil
}

type nvConnection struct {
	log      *zap.Logger
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

func (conn *nvConnection) StageStarting(stage int) {
	conn.log.Info("connection starting", zap.Int("stage", stage))
}

func (conn *nvConnection) StageComplete(stage int) {
	conn.log.Info("connection complete", zap.Int("stage", stage))
}

func (conn *nvConnection) StageFailed(stage int, errorCode int) {
	conn.log.Error("connection failed",
		zap.Int("stage", stage),
		zap.Int("error_code", errorCode))
}

func (conn *nvConnection) ConnectionStarted() {
	conn.log.Info("connection started")
}

func (conn *nvConnection) ConnectionTerminated(errorCode int) {
	conn.log.Info("connection terminated", zap.Int("error_code", errorCode))
}

func (conn *nvConnection) LogMessage(format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	conn.log.Info("nvstream", zap.String("message", s))
}

func (conn *nvConnection) Rumble(controllerNumber, lowFreqMotor, highFreqMotor uint16) {
	conn.log.Debug("rumble on gamepad",
		zap.Uint16("controller", controllerNumber),
		zap.String("low_freq", fmt.Sprintf("%04x", lowFreqMotor)),
		zap.String("high_freq", fmt.Sprintf("%04x", highFreqMotor)))

	// TODO: implement rumble
}

func (conn *nvConnection) ConnectionStatusUpdate(connectionStatus int) {
	conn.log.Info("connection status update", zap.Int("status", connectionStatus))
}

func (conn *nvConnection) SetHDRMode(hdrEnabled bool) {
	conn.log.Info("set hdr mode", zap.Bool("enabled", hdrEnabled))

	// TODO: decoderRenderer.SetHDRMode(hdrEnabled)
}

func (conn *nvConnection) RumbleTriggers(controllerNumber, leftTriggerMotor, rightTriggerMotor uint16) {
	conn.log.Debug("rumble triggers on gamepad",
		zap.Uint16("controller", controllerNumber),
		zap.String("left_trigger", fmt.Sprintf("%04x", leftTriggerMotor)),
		zap.String("right_trigger", fmt.Sprintf("%04x", rightTriggerMotor)))

	// TODO: implement rumble triggers
}

func (conn *nvConnection) SetMotionEventState(controllerNumber uint16, motionType uint8, reportRateHz uint16) {
	conn.log.Info("set motion event state",
		zap.Uint16("controller", controllerNumber),
		zap.Uint8("motion_type", motionType),
		zap.Uint16("report_rate_hz", reportRateHz))

	// TODO: implement motion event state
}

func (conn *nvConnection) SetControllerLED(controllerNumber uint16, r, g, b uint8) {
	conn.log.Info("set controller led color",
		zap.Uint16("controller", controllerNumber),
		zap.Uint8("r", r),
		zap.Uint8("g", g),
		zap.Uint8("b", b))

	// TODO: implement controller LED color
}
