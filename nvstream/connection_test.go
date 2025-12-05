package nvstream

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/flarexio/game/thirdparty/moonlight"
)

func TestStartConnection(t *testing.T) {
	assert := assert.New(t)

	log, err := zap.NewDevelopment()
	if err != nil {
		assert.Fail(err.Error())
		return
	}
	zap.ReplaceGlobals(log)

	defer log.Sync()

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
		if strings.Contains(a.Name, "Steam") {
			app = a
			break
		}
	}

	if (app == NvApp{}) {
		assert.Fail("Steam app not found")
		return
	}

	streamConfig := DefaultStreamConfiguration()
	streamConfig.App = app

	conn, err := NewConnection(http, streamConfig)
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	vs := NewVideoStream()
	as := NewAudioStream()

	moonlight.SetupCallbacks(conn, vs, as)

	ctx := context.Background()
	if err := conn.StartApp(ctx, app); err != nil {
		assert.Fail(err.Error())
		return
	}

	time.Sleep(1 * time.Minute)
}

func TestStopConnection(t *testing.T) {
	assert := assert.New(t)

	http, err := NewHTTP("MyGameClient", "localhost")
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	conn, err := NewConnection(http, nil)
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
