package webson

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
)

// PoolConfig is for creating a pool
type PoolConfig struct {
	Name          string // use for connection apply
	Size          int    // max connections the pool can hold, 0 to be unlimited
	ClientRetry   int    // client retry count
	RetryInterval int    // client retry interval
}

// NodeConfig is for node append in a pool
type NodeConfig struct {
	Name  string // node name, will be a random string if empty
	Group string // node group
}

// actual config for one webson connection after negotiation
type negoSet struct {
	streamable bool
	maxStreams int

	compressable  bool
	compressLevel int
}

type msgConfig struct {
	negotiate *negoSet

	extraMask      []byte
	triggerOnStart bool
	synchronized   bool
}

// DialConfig is for client Dial, combined with general webson Config & client only ClientConfig
type DialConfig struct {
	Config
	ClientConfig
}

// ClientConfig is for client connection config when negotiating with server
type ClientConfig struct {
	url    *simpleUrl
	dialer func() (net.Conn, error)

	UseTLS    bool        // use TLS connection no matter the what's the url
	TLSConfig *tls.Config // TLS config for this connection

	ExtraHeaders map[string]string // extra http headers sent for upgrading
}

// Timeout is the config for all timeouts
type Timeout struct {
	HandshakeTimeout int // max wait time for upgrading handshakes
	PongTimeout      int // max wait time for this side to receive a pong after ping
	CloseTimeout     int // max wait time for other side to send Close after this side send a Close
}

// Config is the programer preferred options
type Config struct {
	HeaderVerify func(http.Header) bool // verify http headers when upgrade connections

	EnableStreams  bool // allow streaming for this connection
	MaxStreams     int  // max streams this side can take. little one will be choosed.
	ChunkSize      int  // max fragment payloa size
	BufferSize     int  // buffer size for reading from connection
	MaxPayloadSize int  // single data frame size limit
	TriggerOnStart bool // message trigger on first fragment
	Synchronize    bool // handlers will be triggered on the main goroutine with the Start

	EnableCompress bool // allow compression for this connection
	CompressLevel  int  // compress level defined in deflate

	Timeout *Timeout // all timeout configs

	PingInterval int // how often to ping the other side

	MagicKey    []byte // private magic key, default magic key will be used if not set
	PrivateMask []byte // extra masking key
	AlwaysMask  bool   // mask message even this is the server side
}

func (c *Config) setup() error {
	if !c.EnableStreams {
		c.MaxStreams = 0
	}
	if c.EnableStreams && c.MaxStreams == 0 {
		c.MaxStreams = DEFAULT_MAX_STREAMS
	}
	if c.MaxStreams > MAX_STREAMS_IN_THEORY {
		return fmt.Errorf("stream size %d exceed max %d", c.MaxStreams, MAX_STREAMS_IN_THEORY)
	}
	if c.PrivateMask != nil && (len(c.PrivateMask) == 0 || len(c.PrivateMask)%4 != 0) {
		return fmt.Errorf("PrivateMask size %d is not multlply of 4", len(c.PrivateMask))
	}
	if c.ChunkSize < DEFAULT_CHUNK_SIZE {
		c.ChunkSize = DEFAULT_CHUNK_SIZE
	}
	if c.BufferSize == 0 {
		c.BufferSize = DEFAULT_BUFFER_SIZE
	}
	if c.Timeout == nil {
		c.Timeout = &Timeout{
			HandshakeTimeout: DEFAULT_TIMEOUT,
			PongTimeout:      DEFAULT_TIMEOUT / 2,
			CloseTimeout:     DEFAULT_TIMEOUT,
		}
	}
	if c.PingInterval == 0 {
		c.PingInterval = DEFAULT_TIMEOUT
	}
	return nil
}
