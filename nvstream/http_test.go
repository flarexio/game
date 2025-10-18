package nvstream

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateSelfSignedCertRSA(t *testing.T) {
	assert := assert.New(t)

	validFor := 20 * 365 * 24 * time.Hour
	keyBits := 2048

	certPEM, keyPEM, err := GenerateSelfSignedCertRSA(validFor, keyBits)
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	fmt.Println("Certificate PEM:\n" + string(certPEM))
	fmt.Println("Key PEM:\n" + string(keyPEM))

	certPEMStr := string(certPEM)

	bs, err := json.Marshal(certPEMStr)
	if err != nil {
		assert.Fail(err.Error())
		return
	}

	escaped := strings.ReplaceAll(string(bs), "/", `\/`)

	fmt.Printf("Certificate PEM JSON:\n%s\n", escaped)

	fmt.Printf("%x", certPEM)
}
