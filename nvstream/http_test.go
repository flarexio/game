package nvstream

import (
	"encoding/json"
	"fmt"
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
