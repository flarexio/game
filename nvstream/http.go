package nvstream

import (
	"context"
	"crypto"
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
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/flarexio/game/thirdparty/moonlight"
)

const (
	DEFAULT_HTTPS_PORT int = 47984
	DEFAULT_HTTP_PORT  int = 47989
)

type NvHTTP interface {
	CertPEM() []byte
	ClientCert() *x509.Certificate
	ServerCert() *x509.Certificate
	SetServerCert(certPEM []byte) error
	Sign(data []byte) ([]byte, error)

	ServerInfo() (*ServerInfoResponse, error)

	AppList() ([]NvApp, error)
	LaunchApp(ctx context.Context, appID int, enableHDR bool) (string, error)
	QuitApp(ctx context.Context) error

	ExecutePairingCommand(ctx context.Context, args map[string]string) (*PairResponse, error)
	ExecutePairingChallenge(ctx context.Context) (*PairResponse, error)
	Unpair() error
}

func NewHTTP(uniqueID string, host string) (NvHTTP, error) {
	if uniqueID == "" {
		uniqueID = "0123456789ABCDEF"
	}

	h := &nvHTTP{
		uniqueID: uniqueID,
		host:     host,
		http:     new(http.Client),
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(homeDir, ".flarex", "game", "certs")
	if err := os.MkdirAll(path, 0700); err != nil {
		return nil, err
	}

	h.path = path

	if err := h.loadClientCertificate(); err != nil {
		return nil, err
	}

	return h, nil
}

type nvHTTP struct {
	uniqueID string
	host     string

	path    string
	keyPEM  []byte
	certPEM []byte

	privkeyKey *rsa.PrivateKey
	clientCert *x509.Certificate
	serverCert *x509.Certificate

	http  *http.Client
	https *http.Client
}

func (h *nvHTTP) loadClientCertificate() error {
	certPEM, keyPEM, err := LoadCertificate(h.path)
	if err != nil {
		validFor := 20 * 365 * 24 * time.Hour
		keyBits := 2048

		certPEM, keyPEM, err = GenerateCertificate(validFor, keyBits)
		if err != nil {
			return err
		}

		if err := SaveCertificate(h.path, certPEM, keyPEM); err != nil {
			return err
		}
	}

	h.certPEM = certPEM
	h.keyPEM = keyPEM

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return errors.New("failed to parse certificate PEM")
	}

	clientCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return err
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return errors.New("failed to parse key PEM")
	}

	privkeyKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return err
	}

	h.clientCert = clientCert
	h.privkeyKey = privkeyKey

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}

	h.https = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	serverCertPath := filepath.Join(h.path, "server.crt")

	serverCertPEM, err := os.ReadFile(serverCertPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return err
	}

	serverCertBlock, _ := pem.Decode(serverCertPEM)
	if serverCertBlock == nil {
		return errors.New("failed to parse PEM block from server certificate")
	}

	serverCert, err := x509.ParseCertificate(serverCertBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	h.serverCert = serverCert
	return nil
}

func (h *nvHTTP) CertPEM() []byte {
	return h.certPEM
}

func (h *nvHTTP) ClientCert() *x509.Certificate {
	return h.clientCert
}

func (h *nvHTTP) ServerCert() *x509.Certificate {
	return h.serverCert
}

func (h *nvHTTP) SetServerCert(certPEM []byte) error {
	certPath := filepath.Join(h.path, "server.crt")

	if err := os.WriteFile(certPath, certPEM, 0600); err != nil {
		return err
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return errors.New("failed to parse PEM block from server certificate")
	}

	serverCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	h.serverCert = serverCert
	return nil
}

func (h *nvHTTP) Sign(data []byte) ([]byte, error) {
	hash := sha256.Sum256(data)

	signature, err := rsa.SignPKCS1v15(rand.Reader, h.privkeyKey, crypto.SHA256, hash[:])
	if err != nil {
		return nil, err
	}

	return signature, nil
}

type ServerInfoResponse struct {
	XMLName                xml.Name `xml:"root"`
	Hostname               string   `xml:"hostname"`
	AppVersion             string   `xml:"appversion"`
	GfeVersion             string   `xml:"GfeVersion"`
	UniqueID               string   `xml:"uniqueid"`
	HttpsPort              int      `xml:"HttpsPort"`
	ExternalPort           int      `xml:"ExternalPort"`
	MaxLumaPixelsHEVC      int      `xml:"MaxLumaPixelsHEVC"`
	MAC                    string   `xml:"mac"`
	LocalIP                string   `xml:"LocalIP"`
	ServerCodecModeSupport int      `xml:"ServerCodecModeSupport"`
	PairStatus             int      `xml:"PairStatus"`
	CurrentGame            int      `xml:"currentgame"`
	State                  string   `xml:"state"`
}

func (resp *ServerInfoResponse) IsPaired() bool {
	return resp.PairStatus == 1
}

func (resp *ServerInfoResponse) Supports4K() bool {
	if resp.GfeVersion == "" || strings.HasPrefix(resp.GfeVersion, "2.") {
		return false
	}

	return true
}

func (h *nvHTTP) ServerInfo() (*ServerInfoResponse, error) {
	values := url.Values{}
	values.Add("uniqueid", h.uniqueID)

	url, err := url.Parse("https://" + h.host + ":" + strconv.Itoa(DEFAULT_HTTPS_PORT) + "/serverinfo")
	if err != nil {
		return nil, err
	}

	url.RawQuery = values.Encode()

	resp, err := h.https.Get(url.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	decoder := xml.NewDecoder(resp.Body)

	var info *ServerInfoResponse
	if err := decoder.Decode(&info); err != nil {
		return nil, err
	}

	return info, nil
}

type AppListResponse struct {
	XMLName    xml.Name `xml:"root"`
	StatusCode int      `xml:"status_code,attr"`
	Apps       []NvApp  `xml:"App"`
}

type NvApp struct {
	XMLName        xml.Name `xml:"App"`
	IsHdrSupported int      `xml:"IsHdrSupported"`
	Name           string   `xml:"AppTitle"`
	ID             int      `xml:"ID"`
}

func (app *NvApp) IsHDRSupported() bool {
	return app.IsHdrSupported == 1
}

func (app *NvApp) String() string {
	var hdrSupport string
	if app.IsHDRSupported() {
		hdrSupport = "Yes"
	} else {
		hdrSupport = "Unknown"
	}

	return "Name: " + app.Name + "\n" +
		"HDR Supported: " + hdrSupport + "\n" +
		"ID: " + strconv.Itoa(app.ID) + "\n"
}

func (h *nvHTTP) AppList() ([]NvApp, error) {
	values := url.Values{}
	values.Add("uniqueid", h.uniqueID)

	url, err := url.Parse("https://" + h.host + ":" + strconv.Itoa(DEFAULT_HTTPS_PORT) + "/applist")
	if err != nil {
		return nil, err
	}

	url.RawQuery = values.Encode()

	resp, err := h.https.Get(url.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	decoder := xml.NewDecoder(resp.Body)

	var appListResp *AppListResponse
	if err := decoder.Decode(&appListResp); err != nil {
		return nil, err
	}

	return appListResp.Apps, nil
}

func (h *nvHTTP) LaunchApp(ctx context.Context, appID int, enableHDR bool) (string, error) {
	stream, ok := ctx.Value(CtxKeyStreamConfiguration).(*StreamConfiguration)
	if !ok {
		return "", errors.New("stream configuration not found in context")
	}

	ri, ok := ctx.Value(CtxKeyRemoteInputAES).(*moonlight.RemoteInputAES)
	if !ok {
		return "", errors.New("remote input AES not found in context")
	}

	values := url.Values{}
	values.Add("uniqueid", h.uniqueID)
	values.Add("appid", strconv.Itoa(appID))

	values.Add("mode", fmt.Sprintf("%dx%dx%d", stream.Width, stream.Height, stream.LaunchRefreshRate))
	values.Add("additionalStates", "1")
	values.Add("sops", "1")

	values.Add("rikey", hex.EncodeToString(ri.Key[:]))
	values.Add("rikeyid", fmt.Sprintf("%d", ri.IV[:]))

	if enableHDR {
		values.Add("hdrMode", "1")
		values.Add("clientHdrCapVersion", "0")
		values.Add("clientHdrCapSupportedFlagsInUint32", "0")
		values.Add("clientHdrCapMetaDataId", "NV_STATIC_METADATA_TYPE_1")
		values.Add("clientHdrCapDisplayData", "0x0x0x0x0x0x0x0x0x0x0")
	}

	if stream.PlayLocalAudio {
		values.Add("localAudioPlayMode", "1")
	} else {
		values.Add("localAudioPlayMode", "0")
	}

	values.Add("surroundAudioInfo", strconv.Itoa(stream.AudioConfiguration.SurroundAudioInfo()))
	values.Add("remoteControllersBitmap", strconv.Itoa(stream.AttachedGamepadMask))
	values.Add("gcmap", strconv.Itoa(stream.AttachedGamepadMask))

	if stream.PersistGamepadAfterDisconnect {
		values.Add("gcpersist", "1")
	} else {
		values.Add("gcpersist", "0")
	}

	values.Add("corever", "1")

	url, err := url.Parse("https://" + h.host + ":" + strconv.Itoa(DEFAULT_HTTPS_PORT) + "/launch")
	if err != nil {
		return "", err
	}

	url.RawQuery = values.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return "", err
	}

	resp, err := h.https.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	var raw struct {
		XMLName     xml.Name `xml:"root"`
		StatusCode  int      `xml:"status_code,attr"`
		SessionURL  string   `xml:"sessionUrl0"`
		GameSession int      `xml:"gamesession"`
	}

	decoder := xml.NewDecoder(resp.Body)
	if err := decoder.Decode(&raw); err != nil {
		return "", err
	}

	if raw.GameSession != 1 {
		return "", errors.New("failed to launch app")
	}

	return raw.SessionURL, nil
}

func (h *nvHTTP) QuitApp(ctx context.Context) error {
	values := url.Values{}
	values.Add("uniqueid", h.uniqueID)

	url, err := url.Parse("https://" + h.host + ":" + strconv.Itoa(DEFAULT_HTTPS_PORT) + "/cancel")
	if err != nil {
		return err
	}

	url.RawQuery = values.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return err
	}

	resp, err := h.https.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	var raw struct {
		XMLName    xml.Name `xml:"root"`
		StatusCode int      `xml:"status_code,attr"`
		Cancel     int      `xml:"cancel"`
	}

	decoder := xml.NewDecoder(resp.Body)
	if err := decoder.Decode(&raw); err != nil {
		return err
	}

	if raw.Cancel != 1 {
		return errors.New("failed to quit app")
	}

	return nil
}

type PairResponse struct {
	XMLName                 xml.Name `xml:"root"`
	StatusCode              int      `xml:"status_code,attr"`
	Paired                  int      `xml:"paired"`
	ServerCert              string   `xml:"plaincert"`
	ServerChallengeResponse string   `xml:"challengeresponse"`
	ServerSecret            string   `xml:"pairingsecret"`
}

func (h *nvHTTP) ExecutePairingCommand(ctx context.Context, args map[string]string) (*PairResponse, error) {
	values := url.Values{}
	values.Add("uniqueid", h.uniqueID)
	values.Add("devicename", "roth")
	values.Add("updateState", "1")

	for k, v := range args {
		values.Add(k, v)
	}

	url, err := url.Parse("http://" + h.host + ":" + strconv.Itoa(DEFAULT_HTTP_PORT) + "/pair")
	if err != nil {
		return nil, err
	}

	url.RawQuery = values.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := h.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("HTTP request failed with status: " + resp.Status)
	}

	decoder := xml.NewDecoder(resp.Body)

	var pairResp *PairResponse
	if err := decoder.Decode(&pairResp); err != nil {
		return nil, err
	}

	return pairResp, nil
}

func (h *nvHTTP) ExecutePairingChallenge(ctx context.Context) (*PairResponse, error) {
	values := url.Values{}
	values.Add("uniqueid", h.uniqueID)
	values.Add("devicename", "roth")
	values.Add("updateState", "1")
	values.Add("phrase", "pairchallenge")

	url, err := url.Parse("http://" + h.host + ":" + strconv.Itoa(DEFAULT_HTTP_PORT) + "/pair")
	if err != nil {
		return nil, err
	}

	url.RawQuery = values.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := h.https.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status: %s", resp.Status)
	}

	decoder := xml.NewDecoder(resp.Body)

	var pairResp *PairResponse
	if err := decoder.Decode(&pairResp); err != nil {
		return nil, err
	}

	return pairResp, nil
}

func (h *nvHTTP) Unpair() error {
	values := url.Values{}
	values.Add("uniqueid", h.uniqueID)

	url, err := url.Parse("http://" + h.host + ":" + strconv.Itoa(DEFAULT_HTTP_PORT) + "/unpair")
	if err != nil {
		return err
	}

	url.RawQuery = values.Encode()

	resp, err := http.Get(url.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("HTTP request failed with status: " + resp.Status)
	}

	return nil
}
