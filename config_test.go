package game

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestConfig(t *testing.T) {
	assert := assert.New(t)

	f, err := os.Open("./config.example.yaml")
	if err != nil {
		assert.Fail(err.Error())
		return
	}
	defer f.Close()

	var cfg *Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		assert.Fail(err.Error())
		return
	}

	assert.Len(cfg.WebRTC.ICEServers, 3)
	assert.Equal(Google, cfg.WebRTC.ICEServers[0].Provider)

	assert.Len(cfg.Streams, 1)

	stream := cfg.Streams[0]
	assert.Equal("stream", stream.Name)

	assert.Len(stream.Origins, 1)

	origin := stream.Origins[0]
	assert.NotNil(origin.Video)

	video := origin.Video
	address := video.Address()
	assert.Equal(address.Scheme, "unix")
	assert.Equal(address.Path, "/tmp/stream/video.sock")
	assert.Equal(video.Codec(), CodecH264)
}
