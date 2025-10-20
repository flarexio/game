package nvstream

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

type ContextKey string

const (
	CtxKeyCSeq          ContextKey = "CSeq"
	CtxKeyClientVersion ContextKey = "ClientVersion"
	CtxKeySessionID     ContextKey = "SessionID"
)

// RTSPClient represents an RTSP client connection
type RTSPClient struct {
	conn          net.Conn
	reader        *bufio.Reader
	rtspURL       string
	host          string
	port          int
	sessionID     string
	seqNum        int
	clientVersion string
}

// NewRTSPClient creates a new RTSP client and connects to the server
func NewRTSPClient(rtspURL string) (*RTSPClient, error) {
	// Parse RTSP URL: rtsp://localhost:48010 or rtsp://localhost:48010?sessionid=xxxxx
	if !strings.HasPrefix(rtspURL, "rtsp://") {
		return nil, fmt.Errorf("invalid RTSP URL: %s", rtspURL)
	}

	urlWithoutScheme := strings.TrimPrefix(rtspURL, "rtsp://")

	// Split host:port and query params
	hostPortAndQuery := strings.Split(urlWithoutScheme, "?")
	hostPort := hostPortAndQuery[0]

	parts := strings.Split(hostPort, ":")

	host := parts[0]
	port := 48010 // default RTSP port
	if len(parts) > 1 {
		var err error
		port, err = strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid port: %s", parts[1])
		}
	}

	// Connect to RTSP server
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RTSP server: %w", err)
	}

	client := &RTSPClient{
		conn:          conn,
		reader:        bufio.NewReader(conn),
		rtspURL:       rtspURL,
		host:          host,
		port:          port,
		seqNum:        1,
		clientVersion: "13", // Match Moonlight's client version
	}

	return client, nil
}

func (c *RTSPClient) SetSessionID(sessionID string) {
	c.sessionID = sessionID
}

type RTSPRequest struct {
	method  string
	target  string
	headers map[string]string
	body    string
}

func NewRTSPRequest(method string, target string) *RTSPRequest {
	return &RTSPRequest{
		method:  method,
		target:  target,
		headers: make(map[string]string),
	}
}

func (req *RTSPRequest) SetBody(body string) {
	req.body = body
}

func (req *RTSPRequest) AddHeader(key, value string) {
	req.headers[key] = value
}

func (req *RTSPRequest) Headers(ctx context.Context) (string, error) {
	seq, ok := ctx.Value(CtxKeyCSeq).(int)
	if !ok {
		return "", fmt.Errorf("missing CSeq in context")
	}

	clientVersion, ok := ctx.Value(CtxKeyClientVersion).(string)
	if !ok {
		return "", fmt.Errorf("missing ClientVersion in context")
	}

	sessionID, _ := ctx.Value(CtxKeySessionID).(string)

	var sb strings.Builder

	// Request line: METHOD target RTSP/1.0
	sb.WriteString(fmt.Sprintf("%s %s RTSP/1.0\r\n", req.method, req.target))

	// CSeq header (sequence number)
	sb.WriteString(fmt.Sprintf("CSeq: %d\r\n", seq))

	// X-GS-ClientVersion header
	sb.WriteString(fmt.Sprintf("X-GS-ClientVersion: %s\r\n", clientVersion))

	// Session header (if session already established)
	if sessionID != "" {
		sb.WriteString(fmt.Sprintf("Session: %s\r\n", sessionID))
	}

	// User-Agent header
	sb.WriteString("User-Agent: FlareX Game Client\r\n")

	for key, value := range req.headers {
		sb.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}

	return sb.String(), nil
}

func (req *RTSPRequest) Payload(ctx context.Context) (string, error) {
	request, err := req.Headers(ctx)
	if err != nil {
		return "", err
	}

	request += "\r\n"

	return request, nil
}

func (c *RTSPClient) doRequest(req *RTSPRequest) (*RTSPResponse, error) {
	// Prepare context with CSeq, ClientVersion, and SessionID
	ctx := context.Background()
	ctx = context.WithValue(ctx, CtxKeyCSeq, c.seqNum)
	ctx = context.WithValue(ctx, CtxKeyClientVersion, c.clientVersion)
	ctx = context.WithValue(ctx, CtxKeySessionID, c.sessionID)

	request, err := req.Payload(ctx)
	if err != nil {
		return nil, err
	}

	c.seqNum++

	// Send request
	if _, err := c.conn.Write([]byte(request)); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Send body if present
	if req.body != "" {
		if _, err := c.conn.Write([]byte(req.body)); err != nil {
			return nil, fmt.Errorf("failed to send request body: %w", err)
		}
	}

	// Read response
	return c.readResponse()
}

// RTSPResponse represents an RTSP response
type RTSPResponse struct {
	StatusCode int
	Status     string
	Headers    map[string]string
	Body       string
}

// readResponse reads and parses an RTSP response
func (c *RTSPClient) readResponse() (*RTSPResponse, error) {
	resp := &RTSPResponse{
		Headers: make(map[string]string),
	}

	// Read status line
	statusLine, err := c.reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read status line: %w", err)
	}

	statusLine = strings.TrimRight(statusLine, "\r\n")

	// Parse status line: RTSP/1.0 200 OK
	parts := strings.SplitN(statusLine, " ", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid status line: %s", statusLine)
	}

	resp.StatusCode, err = strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid status code: %s", parts[1])
	}

	if len(parts) >= 3 {
		resp.Status = parts[2]
	}

	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, fmt.Errorf("failed to read headers: %w", err)
		}

		line = strings.TrimRight(line, "\r\n")

		// Empty line indicates end of headers
		if line == "" {
			break
		}

		headerParts := strings.SplitN(line, ": ", 2)
		if len(headerParts) == 2 {
			key := headerParts[0]
			value := headerParts[1]
			resp.Headers[key] = value
		}
	}

	bodyBytes, err := io.ReadAll(c.reader)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	resp.Body = string(bodyBytes)

	return resp, nil
}

// Options sends an OPTIONS request
func (c *RTSPClient) Options() error {
	req := NewRTSPRequest("OPTIONS", "*")

	resp, err := c.doRequest(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("OPTIONS failed with status: %d %s", resp.StatusCode, resp.Status)
	}

	return nil
}

// Describe sends a DESCRIBE request to get stream information
func (c *RTSPClient) Describe() (*RTSPResponse, error) {
	req := NewRTSPRequest("DESCRIBE", "*")
	req.AddHeader("Accept", "application/sdp")
	req.AddHeader("If-Modified-Since", "Thu, 01 Jan 1970 00:00:00 GMT")

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("DESCRIBE failed with status: %d %s", resp.StatusCode, resp.Status)
	}

	return resp, nil
}

// Setup sends a SETUP request for a stream
func (c *RTSPClient) Setup(streamID string) (*RTSPResponse, error) {
	req := NewRTSPRequest("SETUP", "streamid="+streamID)
	req.AddHeader("Transport", "unicast;X-GS-ClientPort=50000-50001")
	req.AddHeader("If-Modified-Since", "Thu, 01 Jan 1970 00:00:00 GMT")

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("SETUP failed with status: %d %s", resp.StatusCode, resp.Status)
	}

	return resp, nil
}

// Announce sends an ANNOUNCE request with the given SDP body
func (c *RTSPClient) Announce(sdpBody string) (*RTSPResponse, error) {
	req := NewRTSPRequest("ANNOUNCE", "streamid=control/13/0")
	req.AddHeader("Content-Type", "application/sdp")
	req.SetBody(sdpBody)

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("ANNOUNCE failed with status: %d %s", resp.StatusCode, resp.Status)
	}

	return resp, nil
}

// // Play sends a PLAY request to start streaming
// func (c *RTSPClient) Play() error {
// 	resp, err := c.sendRequest("PLAY", "streamid=video", "")
// 	if err != nil {
// 		return err
// 	}

// 	if resp.StatusCode != 200 {
// 		return fmt.Errorf("PLAY failed with status: %d %s", resp.StatusCode, resp.Status)
// 	}

// 	return nil
// }

// GetSessionID returns the current session ID
func (c *RTSPClient) GetSessionID() string {
	return c.sessionID
}

// Close closes the RTSP connection
func (c *RTSPClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
