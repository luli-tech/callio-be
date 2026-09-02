package eslclient

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"strings"
	"sync"
	"time"
)

// Common FreeSWITCH ESL errors.
var (
	ErrNotConnected     = errors.New("esl client is not connected")
	ErrCommandRejected  = errors.New("esl command rejected by freeswitch")
	ErrAuthFailed       = errors.New("esl authentication failed")
)

// Event represents a parsed FreeSWITCH event.
type Event struct {
	Name    string
	Headers map[string]string
	Body    string
}

// Config defines connection details for FreeSWITCH Event Socket.
type Config struct {
	Host     string
	Port     int
	Password string
	Timeout  time.Duration
}

// Client represents a thread-safe connection to FreeSWITCH Event Socket.
type Client struct {
	cfg     Config
	conn    net.Conn
	reader  *textproto.Reader
	writer  *bufio.Writer
	mu      sync.Mutex
	closed  bool
}

// NewClient creates a new FreeSWITCH ESL client.
func NewClient(cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	return &Client{
		cfg: cfg,
	}
}

// Connect establishes TCP connection and performs authentication handshake.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	addr := fmt.Sprintf("%s:%d", c.cfg.Host, c.cfg.Port)
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to dial freeswitch esl at %s: %w", addr, err)
	}

	c.conn = conn
	c.reader = textproto.NewReader(bufio.NewReader(conn))
	c.writer = bufio.NewWriter(conn)
	c.closed = false

	// Read initial auth request header
	header, err := c.reader.ReadMIMEHeader()
	if err != nil {
		_ = c.conn.Close()
		return fmt.Errorf("failed to read initial esl header: %w", err)
	}

	if header.Get("Content-Type") == "auth/request" {
		if err := c.authenticate(); err != nil {
			_ = c.conn.Close()
			return err
		}
	}

	return nil
}

func (c *Client) authenticate() error {
	cmd := fmt.Sprintf("auth %s\n\n", c.cfg.Password)
	if _, err := c.writer.WriteString(cmd); err != nil {
		return err
	}
	if err := c.writer.Flush(); err != nil {
		return err
	}

	header, err := c.reader.ReadMIMEHeader()
	if err != nil {
		return err
	}

	reply := header.Get("Reply-Text")
	if !strings.HasPrefix(reply, "+OK") {
		return fmt.Errorf("%w: %s", ErrAuthFailed, reply)
	}

	return nil
}

// SendAPI executes a blocking API command on FreeSWITCH and returns output.
func (c *Client) SendAPI(ctx context.Context, command, args string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed || c.conn == nil {
		return "", ErrNotConnected
	}

	cmdStr := fmt.Sprintf("api %s %s\n\n", command, args)
	if _, err := c.writer.WriteString(cmdStr); err != nil {
		return "", err
	}
	if err := c.writer.Flush(); err != nil {
		return "", err
	}

	header, err := c.reader.ReadMIMEHeader()
	if err != nil {
		return "", err
	}

	var body string
	contentLenStr := header.Get("Content-Length")
	if contentLenStr != "" {
		var contentLen int
		if _, err := fmt.Sscanf(contentLenStr, "%d", &contentLen); err == nil && contentLen > 0 {
			buf := make([]byte, contentLen)
			if _, err := ioReadFull(c.conn, buf); err == nil {
				body = string(buf)
			}
		}
	}

	return body, nil
}

func ioReadFull(r net.Conn, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := r.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// Close closes the ESL connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed || c.conn == nil {
		return nil
	}

	c.closed = true
	return c.conn.Close()
}

