package webson

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"sync"
	"time"
)

type MessageType int

// predefined message types
const (
	// TextMessage indicates this message is UTF-8 encoded text.
	TextMessage = MessageType(1)

	// BinaryMessage indiacates this message is binary data.
	BinaryMessage = MessageType(2)

	// StreamMessage indicates this message is part of a stream data to multiplex in websocket.
	// StreamMessage is non-standard.
	// Message payload will have 1 bit for stream cancel, 31 bits for stream identifier.
	// It's specially implemented for concurrent message exchange scenario.
	// Inspired by http/2.0
	// Taking some reference from https://datatracker.ietf.org/doc/html/rfc6455#section-5.4
	// StreamMessage = MessageType(4)

	// CloseMessage indicates that side will not send any more data. Stream Message will no
	CloseMessage = MessageType(8)

	// PingMessage indicates this message is a ping.
	PingMessage = MessageType(9)

	// PongMessage indicates this message is a pong.
	PongMessage = MessageType(10)
)

type msgSendOptions struct {
	doCompress    bool
	compressLevel int

	streamlize   bool
	streamId     int
	cancelStream bool

	doMask bool
}

type msgReceivedStatus struct {
	compressed bool

	isStream     bool
	streamId     int
	streamCancel bool

	size int64

	masked       bool
	isFromClient bool

	updateLock sync.Mutex

	msgPool chan []byte

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Message is a complete frame struct.
// Message can't be optimized only by user specify, it's replated to the current connection state.
// It can be created by:
// 1. ReadConnection 2. Message Split
// 3. Dispatch Simple payload 4. Dispatch iter payload
type Message struct {
	send    *msgSendOptions
	receive *msgReceivedStatus
	config  *msgConfig

	isComplete bool
	mask       []byte

	// entity is payload buffer when it's receiving, or raw frame data when it's sending
	entity  bytes.Buffer
	Payload []byte // only used when creating msg
	Type    MessageType
}

func (m *Message) spawnVessel() *Message {

	return &Message{
		config: m.config,
		send:   m.send,

		Type: m.Type,
	}
}

func (m *Message) isControl() bool {
	return int(m.Type) >= 8
}

func (m *Message) setMask(mask []byte) {
	extraMask := m.config.extraMask
	if extraMask == nil {
		m.mask = mask
		return
	}
	m.mask = make([]byte, len(extraMask))
	for i, b := range extraMask {
		m.mask[i] = b ^ mask[i%4]
	}
}

func (m *Message) maskPayload(payload []byte) {
	key := m.mask
	maskSize := len(key)
	for i := range payload {
		payload[i] ^= key[i%maskSize]
	}
}

func (m *Message) assemble() error {
	payload := m.Payload
	msgSize := len(payload)
	if m.send.streamlize {
		msgSize += streamBytes // bytes for stream id
	}
	frameSize := 2 + msgSize // 2 bytes for meta
	if m.send.doMask {
		frameSize += 4 // 4 bytes for mask
	}
	if msgSize >= 65536 {
		frameSize += 8 // 8 bytes for large payload
	} else if msgSize > 125 {
		frameSize += 2 // 2 bytes for normal payload
	}
	var frame = make([]byte, frameSize)

	frame[0] = byte(m.Type)
	if m.isComplete || m.isControl() {
		frame[0] |= 0b1000_0000
	}
	if m.send.doCompress {
		frame[0] |= 0b0100_0000
	}
	if m.send.streamlize {
		frame[0] |= 0b0010_0000
	}
	var instantMask = make([]byte, 0)
	if m.send.doMask {
		frame[1] |= 0b1000_0000
		instantMask = createMask()
		m.setMask(instantMask)
		m.maskPayload(payload)
	}

	pos := 2
	switch {
	case msgSize >= 65536:
		frame[1] |= 127
		pos += 8
		binary.BigEndian.PutUint64(frame[pos:], uint64(msgSize))
	case msgSize > 125:
		frame[1] |= 126
		pos += 2
		binary.BigEndian.PutUint16(frame[pos:], uint16(msgSize))
	default:
		frame[1] |= byte(msgSize)
	}

	if m.send.doMask {
		copy(frame[pos:], instantMask)
		pos += 4
	}
	if m.send.streamlize {
		binary.BigEndian.PutUint16(frame[pos:], uint16(m.send.streamId))
		if m.send.cancelStream {
			frame[pos] |= 0b1000_0000
		}
		pos += streamBytes
	}
	copy(frame[pos:], payload)
	_, e := m.entity.Write(frame)
	return e
}

func (m *Message) parseMeta(raw []byte) error {
	msgType := raw[0] & 0b0000_1111
	fin_ := raw[0]&0b1000_0000 != 0
	rsv1 := raw[0]&0b0100_0000 != 0
	rsv2 := raw[0]&0b0010_0000 != 0
	rsv3 := raw[0]&0b0001_0000 != 0
	mskd := raw[1]&0b1000_0000 != 0
	size := int64(raw[1] & 0b0111_1111)

	if msgType >= 8 {
		if !fin_ {
			return errors.New("control frame is not complete")
		}
		if size > 125 {
			return errors.New("control frame is too large")
		}
	}
	if msgType == 0 && fin_ {
		return errors.New("unrecognized fin or continue")
	}
	if rsv1 && !m.config.negotiate.compressable {
		return errors.New("unrecognized rsv1")
	}
	if rsv2 && !m.config.negotiate.compressable {
		return errors.New("unrecognized rsv2")
	}
	if rsv3 {
		return errors.New("unrecognized rsv3")
	}
	if m.receive.isFromClient && !mskd {
		return errors.New("client msg is not masked")
	}
	m.Type = MessageType(msgType)
	m.isComplete = fin_
	m.receive.compressed = rsv1
	m.receive.isStream = rsv2
	m.receive.masked = mskd
	m.receive.size = size
	if m.receive.isStream && m.receive.size < streamBytes {
		return errors.New("msg size too small to contain stream id")
	}
	return nil
}

func (m *Message) merge(more *Message) error {
	if m.config.triggerOnStart {
		// may block reading from connection
		m.receive.updateLock.Lock()
		defer m.receive.updateLock.Unlock()
	}
	m.receive.UpdatedAt = more.receive.CreatedAt
	m.isComplete = more.isComplete
	if _, e := io.Copy(&m.entity, &more.entity); e != nil {
		return e
	}
	return nil
}

// split msg for write
func (m *Message) split(sizeLimit int) []*Message {
	size := len(m.Payload)
	if size == 0 {
		return []*Message{{
			send:       m.send,
			config:     m.config,
			isComplete: true,
			Type:       m.Type,
		}}
	}
	chunks := size / sizeLimit
	if chunks*sizeLimit < size {
		// math.Ceil
		chunks += 1
	}
	var result = make([]*Message, chunks)
	for i := 0; i < chunks; i++ {
		l, r := i*sizeLimit, (i+1)*sizeLimit
		if r > size {
			r = size
		}
		msg := &Message{
			send:   m.send,
			config: m.config,

			isComplete: i == int(chunks)-1,

			Type:    m.Type,
			Payload: m.Payload[l:r],
		}
		result[i] = msg
	}
	return result
}

func (m *Message) Read() ([]byte, error) {
	if m.config.triggerOnStart {
		m.receive.updateLock.Lock()
		defer m.receive.updateLock.Unlock()
		if !m.isComplete {
			return nil, MsgYetComplete{}
		}
	}
	return io.ReadAll(&m.entity)
}

// ReadIter generate payload chunk by chunk
func (m *Message) ReadIter(chanSize int) <-chan []byte {
	return nil
}
