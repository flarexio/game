package nvstream

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRTSPHandshake(t *testing.T) {
	assert := assert.New(t)

	// First, launch an app to get the RTSP URL
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
		assert.Fail("app not found")
		return
	}

	ctx := context.Background()
	rtspSessionURL, err := http.LaunchApp(ctx, appID, false)
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	fmt.Println("RTSP Session URL: " + rtspSessionURL)

	// Wait a bit for the server to be ready
	time.Sleep(time.Second)

	// Send OPTIONS request
	{
		// Create RTSP client with the session URL
		client, err := NewRTSPClient(rtspSessionURL)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		defer client.Close()

		err = client.Options()
		if err != nil {
			assert.Fail(err.Error())
			return
		}

		fmt.Println("RTSP OPTIONS successful")
		fmt.Println()
	}

	// Send DESCRIBE request
	{
		// Create RTSP client with the session URL
		client, err := NewRTSPClient(rtspSessionURL)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		defer client.Close()

		resp, err := client.Describe()
		if err != nil {
			assert.Fail(err.Error())
			return
		}

		fmt.Println("RTSP DESCRIBE successful")
		fmt.Println("Headers:")
		for key, value := range resp.Headers {
			fmt.Printf("  %s: %s\n", key, value)
		}

		fmt.Println()

		if resp.Body != "" {
			fmt.Println("SDP Body:\n" + resp.Body)
		}
	}

	var sessionID string

	// Send SETUP request for audio track
	{
		// Create RTSP client with the session URL
		client, err := NewRTSPClient(rtspSessionURL)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		defer client.Close()

		resp, err := client.Setup("audio/0/0")
		if err != nil {
			assert.Fail(err.Error())
			return
		}

		fmt.Println("RTSP SETUP audio successful")
		fmt.Println("Headers:")
		for key, value := range resp.Headers {
			if key == "Session" {
				sessionID = strings.SplitN(value, ";", 2)[0]
			}

			fmt.Printf("  %s: %s\n", key, value)
		}

		fmt.Println()

		if resp.Body != "" {
			fmt.Println("Body:\n" + resp.Body)
		}
	}

	// Send SETUP request for video track
	{
		// Create RTSP client with the session URL
		client, err := NewRTSPClient(rtspSessionURL)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		defer client.Close()

		resp, err := client.Setup("video/0/0")
		if err != nil {
			assert.Fail(err.Error())
			return
		}

		fmt.Println("RTSP SETUP video successful")
		fmt.Println("Headers:")
		for key, value := range resp.Headers {
			fmt.Printf("  %s: %s\n", key, value)
		}

		fmt.Println()

		if resp.Body != "" {
			fmt.Println("Body:\n" + resp.Body)
		}
	}

	// Send SETUP request for control track
	{
		// Create RTSP client with the session URL
		client, err := NewRTSPClient(rtspSessionURL)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		defer client.Close()

		resp, err := client.Setup("control/13/0")
		if err != nil {
			assert.Fail(err.Error())
			return
		}

		fmt.Println("RTSP SETUP control successful")
		fmt.Println("Headers:")
		for key, value := range resp.Headers {
			fmt.Printf("  %s: %s\n", key, value)
		}

		fmt.Println()

		if resp.Body != "" {
			fmt.Println("Body:\n" + resp.Body)
		}
	}

	if sessionID == "" {
		assert.Fail("Session ID not found from SETUP response")
		return
	}

	fmt.Println("RTSP Session ID: " + sessionID)

	// Send ANNOUNCE request
	{
		// Create RTSP client with the session URL
		client, err := NewRTSPClient(rtspSessionURL)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		defer client.Close()

		client.SetSessionID(sessionID)

		rdpData := "v=0" + "\r\n" +
			"o=android 0 0 IN IPv4 127.0.0.1\r\n" +
			"s=FlareX Game Client\r\n"

		resp, err := client.Announce(rdpData)
		if err != nil {
			assert.Fail(err.Error())
			return
		}

		fmt.Println("RTSP ANNOUNCE successful")
		fmt.Println("Headers:")
		for key, value := range resp.Headers {
			fmt.Printf("  %s: %s\n", key, value)
		}

		fmt.Println()

		if resp.Body != "" {
			fmt.Println("Body:\n" + resp.Body)
		}
	}
}
