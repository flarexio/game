package nvstream

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPair(t *testing.T) {
	assert := assert.New(t)

	http, err := NewNvHTTP("MyGameClient", "localhost")
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	client := NewPairingManager(http)

	// Client 產生 PIN
	pin := fmt.Sprintf("%04d", rand.Intn(10000))

	fmt.Println("===========================================")
	fmt.Printf("配對 PIN 碼: %s\n", pin)
	fmt.Println("===========================================")
	fmt.Println("步驟：")
	fmt.Println("1. 記住這個 PIN 碼")
	fmt.Println("2. 5 秒後會自動開始配對")
	fmt.Println("3. Sunshine 會彈出配對視窗，請輸入 PIN 碼")
	fmt.Println("===========================================")

	// 等待 5 秒讓使用者準備
	fmt.Println("5 秒後開始配對流程...")

	time.Sleep(5 * time.Second)

	fmt.Println("開始配對...")

	// 執行配對
	state := client.Pair(pin)

	if !assert.Equal(PairStatePaired, state) {
		fmt.Printf("配對失敗，狀態碼: %d\n", state)
		return
	}

	fmt.Println("配對成功！")
}
