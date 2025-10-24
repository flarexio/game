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

		resp, err := client.Options()
		if err != nil {
			assert.Fail(err.Error())
			return
		}

		fmt.Println("RTSP OPTIONS " + resp.Status)
		fmt.Println()

		client.Close()
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

		fmt.Println("RTSP DESCRIBE " + resp.Status)
		fmt.Println("Headers:")
		for key, value := range resp.Headers {
			fmt.Printf("- %s: %s\n", key, value)
		}

		if resp.Body != "" {
			fmt.Println("SDP Body:\n" + resp.Body)
		}

		fmt.Println()

		client.Close()
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

		fmt.Println("RTSP SETUP audio " + resp.Status)
		fmt.Println("Headers:")
		for key, value := range resp.Headers {
			if key == "Session" {
				sessionID = strings.SplitN(value, ";", 2)[0]
			}

			fmt.Printf("- %s: %s\n", key, value)
		}

		fmt.Println()

		client.Close()
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

		fmt.Println("RTSP SETUP video " + resp.Status)
		fmt.Println("Headers:")
		for key, value := range resp.Headers {
			fmt.Printf("- %s: %s\n", key, value)
		}

		fmt.Println()

		client.Close()
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

		fmt.Println("RTSP SETUP control " + resp.Status)
		fmt.Println("Headers:")
		for key, value := range resp.Headers {
			fmt.Printf("- %s: %s\n", key, value)
		}

		fmt.Println()

		client.Close()
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

		var rdpData string

		// SDP Header
		rdpData += "v=0\r\n" +
			"o=android 0 0 IN IPv4 127.0.0.1\r\n" +
			"s=NVIDIA Streaming Client\r\n"

		// SDP Body
		rdpData += "a=x-ml-general.featureFlags:3 \r\n" +
			"a=x-ss-general.encryptionEnabled:1 \r\n" +
			"a=x-ss-video[0].chromaSamplingType:0 \r\n" +
			"a=x-nv-video[0].clientViewportWd:1920 \r\n" +
			"a=x-nv-video[0].clientViewportHt:1080 \r\n" +
			"a=x-nv-video[0].maxFPS:60 \r\n" +
			"a=x-nv-video[0].packetSize:1024 \r\n" +
			"a=x-nv-video[0].rateControlMode:4 \r\n" +
			"a=x-nv-video[0].timeoutLengthMs:7000 \r\n" +
			"a=x-nv-video[0].framesWithInvalidRefThreshold:0 \r\n" +
			"a=x-nv-video[0].initialBitrateKbps:100000 \r\n" +
			"a=x-nv-video[0].initialPeakBitrateKbps:100000 \r\n" +
			"a=x-nv-vqos[0].bw.minimumBitrateKbps:100000 \r\n" +
			"a=x-nv-vqos[0].bw.maximumBitrateKbps:100000 \r\n" +
			"a=x-ml-video.configuredBitrateKbps:100000 \r\n" +
			"a=x-nv-vqos[0].fec.enable:1 \r\n" +
			"a=x-nv-vqos[0].videoQualityScoreUpdateTime:5000 \r\n" +
			"a=x-nv-vqos[0].qosTrafficType:5 \r\n" +
			"a=x-nv-aqos.qosTrafficType:4 \r\n" +
			"a=x-nv-general.featureFlags:167 \r\n" +
			"a=x-nv-general.useReliableUdp:13 \r\n" +
			"a=x-nv-vqos[0].fec.minRequiredFecPackets:2 \r\n" +
			"a=x-nv-vqos[0].bllFec.enable:0 \r\n" +
			"a=x-nv-vqos[0].drc.enable:0 \r\n" +
			"a=x-nv-general.enableRecoveryMode:0 \r\n" +
			"a=x-nv-video[0].videoEncoderSlicesPerFrame:1 \r\n" +
			"a=x-nv-clientSupportHevc:0 \r\n" +
			"a=x-nv-vqos[0].bitStreamFormat:0 \r\n" +
			"a=x-nv-video[0].dynamicRangeMode:0 \r\n" +
			"a=x-nv-video[0].maxNumReferenceFrames:0 \r\n" +
			"a=x-nv-video[0].clientRefreshRateX100:6000 \r\n" +
			"a=x-nv-audio.surround.numChannels:2 \r\n" +
			"a=x-nv-audio.surround.channelMask:3 \r\n" +
			"a=x-nv-audio.surround.enable:0 \r\n" +
			"a=x-nv-audio.surround.AudioQuality:0 \r\n" +
			"a=x-nv-aqos.packetDuration:5 \r\n" +
			"a=x-nv-video[0].encoderCscMode:0 \r\n"

		// SDP Tail
		rdpData += "t=0 0\r\n" +
			"m=video 47998  \r\n"

		resp, err := client.Announce(rdpData)
		if err != nil {
			assert.Fail(err.Error())
			return
		}

		fmt.Println("RTSP ANNOUNCE successful")
		fmt.Println("Headers:")
		for key, value := range resp.Headers {
			fmt.Printf("- %s: %s\n", key, value)
		}

		fmt.Println()

		client.Close()
	}

	{
		// Create RTSP client with the session URL
		client, err := NewRTSPClient(rtspSessionURL)
		if err != nil {
			assert.Fail(err.Error())
			return
		}
		defer client.Close()

		resp, err := client.Play()
		if err != nil {
			assert.Fail(err.Error())
			return
		}

		fmt.Println("RTSP PLAY " + resp.Status)
		fmt.Println("Headers:")
		for key, value := range resp.Headers {
			fmt.Printf("- %s: %s\n", key, value)
		}

		fmt.Println()

		client.Close()
	}
}
