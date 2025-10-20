package nvstream

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerInfo(t *testing.T) {
	assert := assert.New(t)

	http, err := NewNvHTTP("MyGameClient", "localhost")
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	info, err := http.ServerInfo()
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	bs, err := json.MarshalIndent(&info, "", "  ")
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	fmt.Println("Server Info:")
	fmt.Println(string(bs))
}

func TestAppList(t *testing.T) {
	assert := assert.New(t)

	http, err := NewNvHTTP("MyGameClient", "localhost")
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	appList, err := http.AppList()
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	assert.GreaterOrEqual(len(appList), 2)

	for _, app := range appList {
		fmt.Println(app.String())
	}
}

func TestLaunchApp(t *testing.T) {
	assert := assert.New(t)

	http, err := NewNvHTTP("MyGameClient", "localhost")
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	appList, err := http.AppList()
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	var appID int
	for _, app := range appList {
		if strings.HasPrefix(app.Name, "Steam") {
			appID = app.ID
			break
		}
	}

	if appID == 0 {
		assert.Fail("Steam app not found")
		return
	}

	ctx := context.Background()
	rtspSessionURL, err := http.LaunchApp(ctx, appID, false)
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	fmt.Println("RTSP Session URL: " + rtspSessionURL)

	assert.Contains(rtspSessionURL, "rtsp")
}

func TestQuitApp(t *testing.T) {
	assert := assert.New(t)

	http, err := NewNvHTTP("MyGameClient", "localhost")
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	ctx := context.Background()
	if err := http.QuitApp(ctx); err != nil {
		assert.Fail(err.Error())
		return
	}
}
