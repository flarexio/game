package nvstream

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

func GenerateCertificate(validFor time.Duration, keyBits int) (certPEM, keyPEM []byte, err error) {
	if validFor == 0 {
		validFor = 20 * 365 * 24 * time.Hour
	}

	if keyBits == 0 {
		keyBits = 2048
	}

	// 產生 RSA 私鑰
	priv, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return nil, nil, err
	}

	// 憑證資訊
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, err
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(validFor)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "FlareX GameStream Client",
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		SignatureAlgorithm:    x509.SHA256WithRSA,
		BasicConstraintsValid: false,
		IsCA:                  false,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	keyPKCS1 := x509.MarshalPKCS1PrivateKey(priv)
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyPKCS1})

	return certPEM, keyPEM, nil
}

func SaveCertificate(path string, certPEM, keyPEM []byte) error {
	if len(certPEM) == 0 || len(keyPEM) == 0 {
		return errors.New("empty cert or key")
	}

	certPath := filepath.Join(path, "client.crt")

	if err := os.WriteFile(certPath, certPEM, 0600); err != nil {
		return err
	}

	keyPath := filepath.Join(path, "client.key")

	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return err
	}

	return nil
}

func LoadCertificate(path string) (certPEM, keyPEM []byte, err error) {
	certPath := filepath.Join(path, "client.crt")
	keyPath := filepath.Join(path, "client.key")

	certPEM, err = os.ReadFile(certPath)
	if err != nil {
		return nil, nil, err
	}

	keyPEM, err = os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, err
	}

	return certPEM, keyPEM, nil
}
