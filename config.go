package webson

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
)

type PoolConfig struct {
	Size          int // max connections the pool can hold, 0 to be unlimited
	ClientRetry   int // client retry count
	RetryInterval int
}

type NodeConfig struct {
	Name  string
	Group string
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
}

type DialConfig struct {
	Config
	ClientConfig
}

// ClientConfig is for client connection config when negotiating with server
type ClientConfig struct {
	url    *simpleUrl
	dialer func() (net.Conn, error)

	UseTLS    bool
	TLSConfig *tls.Config

	ExtraHeaders map[string]string
}

type Timeout struct {
	HandshakeTimeout int
	PongTimeout      int
}

// Config is the programer preferred options
type Config struct {
	HeaderVerify func(http.Header) bool

	EnableStreams  bool
	MaxStreams     int
	ChunkSize      int
	BufferSize     int
	MaxPayloadSize int // single data frame size limit

	EnableCompress bool
	CompressLevel  int

	Timeout *Timeout

	PingInterval   int
	TriggerOnStart bool

	MagicKey    []byte
	PrivateMask []byte
	AlwaysMask  bool
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
	if c.BufferSize < DEFAULT_CHUNK_SIZE {
		c.BufferSize = DEFAULT_CHUNK_SIZE
	}
	if c.Timeout == nil {
		c.Timeout = &Timeout{
			HandshakeTimeout: DEFAULT_TIMEOUT,
			PongTimeout:      DEFAULT_TIMEOUT,
		}
	}
	if c.PingInterval == 0 {
		c.PingInterval = DEFAULT_TIMEOUT / 2
	}
	return nil
}
