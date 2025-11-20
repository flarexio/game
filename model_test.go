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

	assert.Len(cfg.Streams, 2)

	{
		stream := cfg.Streams[0]
		assert.Equal(TransportNV, stream.Transport)
		assert.Equal("https://localhost:47984", stream.Address.String())

		assert.Equal(CodecH264, stream.Video.Codec())
		assert.Equal(CodecOpus, stream.Audio.Codec())
	}

	{
		stream := cfg.Streams[1]
		assert.Equal(TransportRaw, stream.Transport)

		assert.Equal(CodecH264, stream.Video.Codec())
		assert.Equal("unix", stream.Video.Address().Scheme)
		assert.Equal("/tmp/stream/video.sock", stream.Video.Address().Path)

		assert.Equal(CodecOpus, stream.Audio.Codec())
		assert.Equal("unix", stream.Audio.Address().Scheme)
		assert.Equal("/tmp/stream/audio.sock", stream.Audio.Address().Path)
	}
}
