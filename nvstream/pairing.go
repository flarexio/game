package nvstream

import (
	"crypto"
	"crypto/aes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
)

type PairResponse struct {
	XMLName                 xml.Name `xml:"root"`
	Paired                  int      `xml:"paired"`
	PairState               int      `xml:"pairstate"`
	ServerCert              string   `xml:"plaincert"`
	ServerChallengeResponse string   `xml:"challengeresponse"`
	ServerSecret            string   `xml:"pairingsecret"`
}

const (
	DEFAULT_HTTPS_PORT int = 47984
	DEFAULT_HTTP_PORT  int = 47989
)

type PairingClient struct {
	host       string
	uniqueID   string
	deviceName string
	keyPEM     []byte
	certPEM    []byte
	privateKey *rsa.PrivateKey
	clientCert *x509.Certificate
	serverCert *x509.Certificate
}

func NewPairingClient(host string, uniqueID, deviceName string, certPEM, keyPEM []byte) (*PairingClient, error) {
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, errors.New("failed to parse client private key PEM")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse client private key: %w", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, errors.New("failed to parse client certificate PEM")
	}

	clientCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse client certificate: %w", err)
	}

	return &PairingClient{
		host:       host,
		uniqueID:   uniqueID,
		deviceName: deviceName,
		keyPEM:     keyPEM,
		certPEM:    certPEM,
		privateKey: privateKey,
		clientCert: clientCert,
		serverCert: nil,
	}, nil
}

func (pc *PairingClient) Pair(pin string) error {
	// Generate a salt for hashing the PIN
	salt, err := generateRandomBytes(16)
	if err != nil {
		return err
	}

	// Combine the salt and pin, then create an AES key from them
	saltedPin := append(salt, []byte(pin)...)
	aesKey := generateAESKey(saltedPin)

	// Send the salt and get the server cert
	if err := pc.fetchServerCert(salt); err != nil {
		return err
	}

	// Generate a random challenge and encrypt it with our AES key
	randomChallenge, err := generateRandomBytes(16)
	if err != nil {
		return err
	}

	encryptedChallenge, err := encrypt(randomChallenge, aesKey)
	if err != nil {
		return err
	}

	// Send the encrypted challenge to the server
	encryptedServerChallengeResponse, err := pc.sendClientChallenge(encryptedChallenge)
	if err != nil {
		return err
	}

	// Decode the server's response and subsequent challenge
	serverChallengeResponse, err := decrypt(encryptedServerChallengeResponse, aesKey)
	if err != nil {
		return err
	}

	serverResponse := serverChallengeResponse[:sha256.Size]
	serverChallenge := serverChallengeResponse[sha256.Size:48]

	// Using another 16 bytes secret, compute a challenge response hash using the secret, our cert sig, and the challenge
	clientSecret, err := generateRandomBytes(16)
	if err != nil {
		return err
	}

	challengeRespHash := sha256.Sum256(append(append(serverChallenge, pc.clientCert.Signature...), clientSecret...))

	challengeRespEncrypted, err := encrypt(challengeRespHash[:], aesKey)
	if err != nil {
		return err
	}

	// Get the server's signed secret
	serverSecretResp, err := pc.sendServerChallengeResponse(challengeRespEncrypted)
	if err != nil {
		return err
	}

	serverSecret := serverSecretResp[:16]
	serverSignature := serverSecretResp[16:]

	// Ensure the authenticity of the data
	if err := pc.verifyServerSecretSignature(serverSecret, serverSignature); err != nil {
		return err
	}

	// Ensure the server challenge matched what we expected (aka the PIN was correct)
	serverChallengeHash := sha256.Sum256(append(append(randomChallenge, pc.serverCert.Signature...), serverSecret...))
	if !slices.Equal(serverChallengeHash[:], serverResponse) {
		return errors.New("server challenge verification failed (incorrect PIN?)")
	}

	// Send the server our signed secret
	if err := pc.sendClientSignedSecret(clientSecret); err != nil {
		return err
	}

	return pc.pairingChallenge()
}

func (pc *PairingClient) fetchServerCert(salt []byte) error {
	params := url.Values{}
	params.Add("uniqueid", pc.uniqueID)
	params.Add("devicename", pc.deviceName)
	params.Add("updateState", "1")
	params.Add("phrase", "getservercert")
	params.Add("salt", hex.EncodeToString(salt))
	params.Add("clientcert", hex.EncodeToString(pc.certPEM))

	resp, err := pc.doRequest(params)
	if err != nil {
		return err
	}

	if resp.Paired != 1 {
		return errors.New("pairing failed")
	}

	if resp.ServerCert == "" {
		return errors.New("server certificate is empty")
	}

	certBytes, err := hex.DecodeString(resp.ServerCert)
	if err != nil {
		return err
	}

	block, _ := pem.Decode(certBytes)
	if block == nil {
		return errors.New("failed to parse PEM block from server certificate")
	}

	serverCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	pc.serverCert = serverCert

	return nil
}

func (pc *PairingClient) sendClientChallenge(challenge []byte) ([]byte, error) {
	params := url.Values{}
	params.Add("uniqueid", pc.uniqueID)
	params.Add("devicename", pc.deviceName)
	params.Add("updateState", "1")
	params.Add("clientchallenge", hex.EncodeToString(challenge))

	resp, err := pc.doRequest(params)
	if err != nil {
		return nil, err
	}

	if resp.Paired != 1 {
		return nil, errors.New("pairing failed")
	}

	if resp.ServerChallengeResponse == "" {
		return nil, errors.New("server challenge response is empty")
	}

	return hex.DecodeString(resp.ServerChallengeResponse)
}

func (pc *PairingClient) sendServerChallengeResponse(response []byte) ([]byte, error) {
	params := url.Values{}
	params.Add("uniqueid", pc.uniqueID)
	params.Add("devicename", pc.deviceName)
	params.Add("updateState", "1")
	params.Add("serverchallengeresp", hex.EncodeToString(response))

	resp, err := pc.doRequest(params)
	if err != nil {
		return nil, err
	}

	if resp.Paired != 1 {
		return nil, errors.New("pairing failed")
	}

	return hex.DecodeString(resp.ServerSecret)
}

func (pc *PairingClient) verifyServerSecretSignature(serverSecret, signature []byte) error {
	hash := sha256.Sum256(serverSecret)

	publicKey, ok := pc.serverCert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return errors.New("failed to parse server public key")
	}

	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], signature); err != nil {
		return fmt.Errorf("verify server signature: %w", err)
	}

	return nil
}

func (pc *PairingClient) sendClientSignedSecret(clientSecret []byte) error {
	hash := sha256.Sum256(clientSecret)

	signature, err := rsa.SignPKCS1v15(rand.Reader, pc.privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return err
	}

	clientPairingSecret := append(clientSecret, signature...)

	params := url.Values{}
	params.Add("uniqueid", pc.uniqueID)
	params.Add("devicename", pc.deviceName)
	params.Add("updateState", "1")
	params.Add("clientpairingsecret", hex.EncodeToString(clientPairingSecret))

	resp, err := pc.doRequest(params)
	if err != nil {
		return err
	}

	if resp.Paired != 1 {
		return errors.New("pairing failed")
	}

	return nil
}

func (pc *PairingClient) pairingChallenge() error {
	keyPair, err := tls.X509KeyPair(pc.certPEM, pc.keyPEM)
	if err != nil {
		return fmt.Errorf("failed to create key pair: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{keyPair},
		InsecureSkipVerify: true,
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	params := url.Values{}
	params.Add("uniqueid", pc.uniqueID)
	params.Add("devicename", pc.deviceName)
	params.Add("updateState", "1")
	params.Add("phrase", "pairchallenge")

	url := fmt.Sprintf("https://%s:%d/pair?%s", pc.host, DEFAULT_HTTPS_PORT, params.Encode())

	fmt.Printf("Request URL: %s\n", url)

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	fmt.Printf("Response Status: %d\n", resp.StatusCode)
	fmt.Printf("Response Body: %s\n", string(body))

	var pairResp PairResponse
	if err := xml.Unmarshal(body, &pairResp); err != nil {
		return fmt.Errorf("parse response: %w\n%s", err, string(body))
	}

	if pairResp.Paired != 1 {
		return errors.New("pairing failed")
	}

	return nil
}

func (pc *PairingClient) doRequest(params url.Values) (*PairResponse, error) {
	url := fmt.Sprintf("http://%s:%d/pair?%s", pc.host, DEFAULT_HTTP_PORT, params.Encode())

	fmt.Printf("Request URL: %s\n", url)

	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Response Status: %d\n", resp.StatusCode)
	fmt.Printf("Response Body: %s\n", string(body))

	var pairResp PairResponse
	if err := xml.Unmarshal(body, &pairResp); err != nil {
		return nil, fmt.Errorf("parse response: %w\n%s", err, string(body))
	}

	return &pairResp, nil
}

func generateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	return b, err
}

func generateAESKey(keyData []byte) []byte {
	hash := sha256.Sum256(keyData)
	return hash[:16]
}

func encrypt(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()

	// Block rounding: round up to the nearest block size
	blockRoundedSize := (len(data) + blockSize - 1) & ^(blockSize - 1)

	blockRoundedInput := make([]byte, blockRoundedSize)
	copy(blockRoundedInput, data)

	// Encrypt the data
	ciphertext := make([]byte, blockRoundedSize)
	for i := 0; i < blockRoundedSize; i += blockSize {
		block.Encrypt(ciphertext[i:i+blockSize], blockRoundedInput[i:i+blockSize])
	}

	return ciphertext, nil
}

func decrypt(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()

	// Block rounding: round up to the nearest block size
	blockRoundedSize := (len(data) + blockSize - 1) & ^(blockSize - 1)

	blockRoundedInput := make([]byte, blockRoundedSize)
	copy(blockRoundedInput, data)

	// Decrypt the data
	plaintext := make([]byte, blockRoundedSize)
	for i := 0; i < blockRoundedSize; i += blockSize {
		block.Decrypt(plaintext[i:i+blockSize], blockRoundedInput[i:i+blockSize])
	}

	return plaintext, nil
}
