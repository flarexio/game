package nvstream

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPair(t *testing.T) {
	assert := assert.New(t)

	validFor := 20 * 365 * 24 * time.Hour
	keyBits := 2048

	// 產生憑證
	certPEM, keyPEM, err := GenerateSelfSignedCertRSA(validFor, keyBits)
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	// 產生 unique ID
	hash := sha256.Sum256(certPEM)
	uniqueID := strings.ToUpper(hex.EncodeToString(hash[:16]))

	fmt.Println("Unique ID: " + uniqueID)

	// 建立配對客戶端
	client, err := NewPairingClient(
		"localhost",
		uniqueID,
		"MyGameClient",
		certPEM,
		keyPEM,
	)

	if err != nil {
		assert.Fail(err.Error())
		return
	}

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
	if err := client.Pair(pin); err != nil {
		assert.Fail(err.Error())
		return
	}

	fmt.Println("✓ 配對成功!")
}
