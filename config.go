package webson

import (
	"fmt"
	"net/http"
)

// actual config for one webson connection after negotiation
type negotiateConfig struct {
	streamable bool

	compressable  bool
	compressLevel int
}

type msgConfig struct {
	negotiate *negotiateConfig

	extraMask      []byte
	triggerOnStart bool
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

	PingInterval    int
	TriggerInterval int
	TriggerOnStart  bool

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
		c.Timeout = &Timeout{}
		c.Timeout.ResetDefaultTimeout()
	}
	if c.PingInterval <= 0 {
		c.PingInterval = DEFAULT_TIMEOUT / 2
	}
	return nil
}
