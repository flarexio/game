package nvstream

import (
	"context"
	"crypto"
	"crypto/aes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"time"
)

type PairState int

const (
	PairStateNotPaired PairState = iota
	PairStatePaired
	PairStatePinWrong
	PairStateFailed
	PairStateAlreadyInProgress
)

var (
	ErrPairingFailed     = errors.New("pairing failed")
	ErrPairingInProgress = errors.New("pairing already in progress")
)

type PairingManager interface {
	Pair(pin string) PairState
}

type pairingManager struct {
	http NvHTTP
}

func NewPairingManager(http NvHTTP) PairingManager {
	return &pairingManager{
		http: http,
	}
}

func (pm *pairingManager) Pair(pin string) PairState {
	defer pm.http.Unpair()

	// Generate a salt for hashing the PIN
	salt, err := generateRandomBytes(16)
	if err != nil {
		return PairStateFailed
	}

	// Combine the salt and pin, then create an AES key from them
	saltedPin := append(salt, []byte(pin)...)
	aesKey := generateAESKey(saltedPin)

	// Send the salt and get the server cert
	serverCertPEM, err := pm.getServerCert(salt)
	if err != nil {
		if errors.Is(err, ErrPairingInProgress) {
			return PairStateAlreadyInProgress
		}

		return PairStateFailed
	}

	pm.http.SetServerCert(serverCertPEM)

	// Generate a random challenge and encrypt it with our AES key
	randomChallenge, err := generateRandomBytes(16)
	if err != nil {
		return PairStateFailed
	}

	encryptedChallenge, err := encrypt(randomChallenge, aesKey)
	if err != nil {
		return PairStateFailed
	}

	// Send the encrypted challenge to the server
	encryptedServerChallengeResponse, err := pm.sendClientChallenge(encryptedChallenge)
	if err != nil {
		return PairStateFailed
	}

	// Decode the server's response and subsequent challenge
	serverChallengeResponse, err := decrypt(encryptedServerChallengeResponse, aesKey)
	if err != nil {
		return PairStateFailed
	}

	serverResponse := serverChallengeResponse[:sha256.Size]
	serverChallenge := serverChallengeResponse[sha256.Size:48]

	// Using another 16 bytes secret, compute a challenge response hash using the secret, our cert sig, and the challenge
	clientSecret, err := generateRandomBytes(16)
	if err != nil {
		return PairStateFailed
	}

	challengeRespHash := sha256.Sum256(append(append(serverChallenge, pm.http.ClientCert().Signature...), clientSecret...))

	challengeRespEncrypted, err := encrypt(challengeRespHash[:], aesKey)
	if err != nil {
		return PairStateFailed
	}

	// Get the server's signed secret
	serverSecretResp, err := pm.sendServerChallengeResponse(challengeRespEncrypted)
	if err != nil {
		return PairStateFailed
	}

	serverSecret := serverSecretResp[:16]
	serverSignature := serverSecretResp[16:]

	// Ensure the authenticity of the data
	if err := pm.verifyServerSecretSignature(serverSecret, serverSignature); err != nil {
		return PairStateFailed
	}

	// Ensure the server challenge matched what we expected (aka the PIN was correct)
	serverChallengeHash := sha256.Sum256(append(append(randomChallenge, pm.http.ServerCert().Signature...), serverSecret...))
	if !slices.Equal(serverChallengeHash[:], serverResponse) {
		return PairStatePinWrong
	}

	// Send the server our signed secret
	if err := pm.sendClientSignedSecret(clientSecret); err != nil {
		return PairStateFailed
	}

	if err := pm.pairingChallenge(); err != nil {
		return PairStateFailed
	}

	return PairStatePaired
}

func (pm *pairingManager) getServerCert(salt []byte) ([]byte, error) {
	args := make(map[string]string)
	args["phrase"] = "getservercert"
	args["salt"] = hex.EncodeToString(salt)
	args["clientcert"] = hex.EncodeToString(pm.http.CertPEM())

	ctx := context.Background()
	resp, err := pm.http.ExecutePairingCommand(ctx, args)
	if err != nil {
		return nil, err
	}

	if resp.Paired != 1 {
		return nil, ErrPairingFailed
	}

	if resp.ServerCert == "" {
		return nil, ErrPairingInProgress
	}

	certBytes, err := hex.DecodeString(resp.ServerCert)
	if err != nil {
		return nil, err
	}

	return certBytes, nil
}

func (pm *pairingManager) sendClientChallenge(challenge []byte) ([]byte, error) {
	args := make(map[string]string)
	args["clientchallenge"] = hex.EncodeToString(challenge)

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5000*time.Millisecond)
	defer cancel()

	resp, err := pm.http.ExecutePairingCommand(ctx, args)
	if err != nil {
		return nil, err
	}

	if resp.Paired != 1 {
		return nil, ErrPairingFailed
	}

	if resp.ServerChallengeResponse == "" {
		return nil, errors.New("server challenge response is empty")
	}

	return hex.DecodeString(resp.ServerChallengeResponse)
}

func (pm *pairingManager) sendServerChallengeResponse(response []byte) ([]byte, error) {
	args := make(map[string]string)
	args["serverchallengeresp"] = hex.EncodeToString(response)

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5000*time.Millisecond)
	defer cancel()

	resp, err := pm.http.ExecutePairingCommand(ctx, args)
	if err != nil {
		return nil, err
	}

	if resp.Paired != 1 {
		return nil, ErrPairingFailed
	}

	return hex.DecodeString(resp.ServerSecret)
}

func (pm *pairingManager) verifyServerSecretSignature(serverSecret, signature []byte) error {
	hash := sha256.Sum256(serverSecret)

	publicKey, ok := pm.http.ServerCert().PublicKey.(*rsa.PublicKey)
	if !ok {
		return errors.New("failed to parse server public key")
	}

	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], signature); err != nil {
		return fmt.Errorf("verify server signature: %w", err)
	}

	return nil
}

func (pm *pairingManager) sendClientSignedSecret(clientSecret []byte) error {
	signature, err := pm.http.Sign(clientSecret)
	if err != nil {
		return err
	}

	clientPairingSecret := append(clientSecret, signature...)

	args := make(map[string]string)
	args["clientpairingsecret"] = hex.EncodeToString(clientPairingSecret)

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5000*time.Millisecond)
	defer cancel()

	resp, err := pm.http.ExecutePairingCommand(ctx, args)
	if err != nil {
		return err
	}

	if resp.Paired != 1 {
		return ErrPairingFailed
	}

	return nil
}

func (pm *pairingManager) pairingChallenge() error {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 5000*time.Millisecond)
	defer cancel()

	resp, err := pm.http.ExecutePairingChallenge(ctx)
	if err != nil {
		return err
	}

	if resp.Paired != 1 {
		return ErrPairingFailed
	}

	return nil
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
