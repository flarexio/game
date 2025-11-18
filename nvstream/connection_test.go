package nvstream

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/flarexio/game/thirdparty/moonlight"
)

func TestStartConnection(t *testing.T) {
	assert := assert.New(t)

	http, err := NewHTTP("MyGameClient", "localhost")
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	appList, err := http.AppList()
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	var app NvApp
	for _, a := range appList {
		if strings.HasPrefix(a.Name, "Steam") {
			app = a
			break
		}
	}

	if (app == NvApp{}) {
		assert.Fail("Steam app not found")
		return
	}

	streamConfig := &StreamConfiguration{
		App:                           app,
		Width:                         1280,
		Height:                        720,
		RefreshRate:                   60,
		LaunchRefreshRate:             60,
		ClientRefreshRateX100:         6000,
		Bitrate:                       1024,
		SOPS:                          true,
		EnableAdaptiveResolution:      false,
		PlayLocalAudio:                false,
		MaxPacketSize:                 1392,
		Remote:                        moonlight.STREAM_CFG_AUTO,
		AudioConfiguration:            moonlight.AUDIO_CONFIGURATION_STEREO,
		SupportedVideoFormats:         moonlight.VIDEO_FORMAT_H264,
		AttachedGamepadMask:           0,
		EncryptionFlags:               moonlight.ENCFLG_ALL,
		ColorRange:                    moonlight.COLOR_RANGE_LIMITED,
		ColorSpace:                    moonlight.COLORSPACE_REC_601,
		PersistGamepadAfterDisconnect: false,
	}

	conn, err := NewConnection("localhost", "MyGameClient", streamConfig)
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	vr := NewVideoReader()
	ar := NewAudioReader()

	moonlight.SetupCallbacks(conn, vr, ar)

	ctx := context.Background()
	if err := conn.StartApp(ctx, app); err != nil {
		assert.Fail(err.Error())
		return
	}

	time.Sleep(1 * time.Minute)
}

func TestStopConnection(t *testing.T) {
	assert := assert.New(t)

	conn, err := NewConnection("localhost", "MyGameClient", nil)
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	ctx := context.Background()
	if err := conn.StopApp(ctx); err != nil {
		assert.Fail(err.Error())
		return
	}
}
