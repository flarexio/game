package game

import (
	"fmt"
	"os"
	"testing"

	"github.com/test-go/testify/assert"
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

	fmt.Println(cfg)
}
