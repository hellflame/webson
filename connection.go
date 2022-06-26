package webson

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

type Adapter interface {
	Close() error
	Connect(string, *Config) error
	Ping() error
	Pong() error
	Dispatch(*Message) error
}

type EventHandler interface {
	OnStatus(Status, Adapter)
	OnMessage(*Message, Adapter)
}

type Connection struct {
	rawConnection net.Conn

	pendingStreams map[int]*Message
	freeStreams    []int
	lastStream     int
	streamIdLock   sync.Mutex

	isClient   bool
	status     Status
	statusLock sync.Mutex
	writeLock  sync.Mutex

	config *Config
	negotiateConfig

	statusEventMap   map[Status]func(Status, Adapter)
	messageEventMap  map[MessageType]func(*Message, Adapter)
	statusEventPool  []func(Status, Adapter)
	messageEventPool []func(*Message, Adapter)
}

func (con *Connection) prepare() {
	config := con.config

	con.rawConnection.SetDeadline(time.Time{})

	// bind default pong for ping
	con.OnMessage(PingMessage, func(m *Message, a Adapter) {
		a.Pong()
	})
	con.OnStatus(StatusReady, func(s Status, a Adapter) {
		for {
			if e := a.Ping(); e != nil {
				break
			}
			time.Sleep(time.Duration(config.PingInterval) * time.Second)
		}
	})

	con.status = StatusYetReady
	con.statusEventMap = make(map[Status]func(Status, Adapter))
	con.messageEventMap = make(map[MessageType]func(*Message, Adapter))

	// if not streamable, pendingStreams is for continue frames
	con.pendingStreams = make(map[int]*Message)
}

func (con *Connection) negotiate() {

}

func (con *Connection) Close() error {
	con.updateStatus(StatusClosed)
	con.Dispatch(&Message{Type: CloseMessage})
	return con.rawConnection.Close()
}

func (con *Connection) Connect(addr string, c *Config) error {
	return nil
}

func (con *Connection) Ping() error {
	return con.Dispatch(&Message{Type: PingMessage})
}

func (con *Connection) Pong() error {
	return con.Dispatch(&Message{Type: PongMessage})
}

func (con *Connection) Dispatch(m *Message) error {
	if e := con.patchMsg(m); e != nil {
		return e
	}
	if !con.streamable {
		con.writeLock.Lock()
		defer con.writeLock.Unlock()
	}

	for _, msg := range m.split(int64(con.config.MaxPayloadSize)) {
		if e := con.writeSingleFrame(msg); e != nil {
			return e
		}
	}
	return nil
}

func (con *Connection) DispatchReader(t MessageType, r io.Reader) (e error) {
	msg := &Message{
		Type: t,
	}
	con.patchMsg(msg)
	if !con.streamable {
		con.writeLock.Lock()
		defer con.writeLock.Unlock()
	}
	chunkSize := con.config.ChunkSize
	var vessel = make([]byte, chunkSize)
	for {
		n, e := r.Read(vessel)
		if e != nil {
			msg.send.cancelStream = true
			con.writeSingleFrame(msg)
			break
		}
		stream := msg.spawnVessel()
		stream.Payload = vessel[0:n]
		e = con.writeSingleFrame(stream)
		if e != nil || n < chunkSize {
			break
		}
	}
	return
}

// patchMsg keeps write Message intact & correct
func (con *Connection) patchMsg(m *Message) error {
	if con.streamable {
		con.streamIdLock.Lock()
		con.lastStream += 1
		if con.lastStream > con.config.MaxStreams {
			con.lastStream = 1
		}
		m.send.streamId = con.lastStream
		con.streamIdLock.Unlock()
	}
	return nil
}

func (con *Connection) writeSingleFrame(m *Message) error {
	status := con.status
	if status == StatusClosed {
		return CantWrite{}
	}
	if status != StatusReady {
		return CantWriteYet{status}
	}
	if e := m.assemble(); e != nil {
		return e
	}
	if con.streamable {
		con.writeLock.Lock()
		defer con.writeLock.Unlock()
	}
	_, e := io.Copy(con.rawConnection, &m.entity)
	return e
}

func (con *Connection) Apply(h EventHandler) {
	con.statusEventPool = append(con.statusEventPool, h.OnStatus)
	con.messageEventPool = append(con.messageEventPool, h.OnMessage)
}

func (con *Connection) OnStatus(s Status, action func(Status, Adapter)) {
	con.statusEventMap[s] = action
}

func (con *Connection) OnMessage(t MessageType, action func(*Message, Adapter)) {
	con.messageEventMap[t] = action
}

func (con *Connection) updateStatus(s Status) {
	con.statusLock.Lock()
	defer con.statusLock.Unlock()

	prevStatus := con.status
	if prevStatus == s {
		// prevent same event keep triggering
		// mainly to prevent close & timeout repeatly triggers
		return
	}
	con.status = s
	if action, ok := con.statusEventMap[s]; ok {
		go action(prevStatus, con)
	}
	for _, action := range con.statusEventPool {
		go action(prevStatus, con)
	}
}

func (con *Connection) triggerMessage(m *Message) {
	if action, ok := con.messageEventMap[m.Type]; ok {
		go action(m, con)
	}
	for _, action := range con.messageEventPool {
		go action(m, con)
	}
}

func (con *Connection) Start() error {
	raw := bufio.NewReaderSize(con.rawConnection, con.config.BufferSize)
	defer con.Close()

	// con.updateStatus(StatusReady)

	triggerOnStart := con.config.TriggerOnStart
	var vessel2 = make([]byte, 2)
	var vessel4 = make([]byte, 4)
	var vessel8 = make([]byte, 8)
	for {
		if s, e := raw.Read(vessel2); e != nil || s != 2 {
			return e
		}
		msg := &Message{
			config: &msgConfig{
				negotiate:      &con.negotiateConfig,
				extraMask:      con.config.PrivateMask,
				triggerOnStart: triggerOnStart,
			},
			receive: &msgReceivedStatus{
				CreatedAt:    time.Now(),
				isFromClient: !con.isClient,
			},
		}
		if e := msg.parseMeta(vessel2); e != nil {
			// con.Dispatch(CloseMessage, )
			return e
		}
		if msg.receive.size == 126 {
			if s, e := raw.Read(vessel2); e != nil || s != 2 {
				return errors.New("msg size not given")
			}
			msg.receive.size = int64(binary.BigEndian.Uint16(vessel2))
		} else if msg.receive.size == 127 {
			if s, e := raw.Read(vessel8); e != nil || s != 8 {
				return errors.New("msg size not given")
			}
			msg.receive.size = int64(binary.BigEndian.Uint64(vessel8))
		}

		if msg.receive.masked {
			if s, e := raw.Read(vessel4); e != nil || s != 4 {
				return errors.New("mask key not given")
			}
			msg.setMask(vessel4)
		}

		if msg.receive.size > 0 {
			if con.config.MaxPayloadSize != 0 && msg.receive.size > int64(con.config.MaxPayloadSize) {
				return MsgTooLarge{}
			}
			payload, e := raw.Peek(int(msg.receive.size))
			if e != nil {
				return e
			}
			raw.Discard(len(payload))
			if msg.receive.masked {
				msg.maskPayload(payload)
			}
			if msg.receive.compressed {

			}
			if msg.receive.isStream {
				cancel := payload[0]&0b1000_0000 != 0
				payload[0] = payload[0] & 0b0111_1111
				msg.receive.streamId = int(binary.BigEndian.Uint64(payload[:streamBytes]))
				if msg.receive.streamId == 0 {
					return errors.New("invalid stream id")
				}
				if cancel {
					delete(con.pendingStreams, msg.receive.streamId)
				}
				payload = payload[streamBytes:]
			}
			if _, e := msg.entity.Write(payload); e != nil {
				return e
			}
		}

		if msg.isControl() {
			con.triggerMessage(msg)
			if msg.Type == CloseMessage {
				// close message will be sent in defer function
				return nil
			}
		} else {
			// no matter streaming or not
			if pending, exist := con.pendingStreams[msg.receive.streamId]; exist {
				if e := pending.merge(msg); e != nil {
					return e
				}
				if msg.isComplete {
					if !triggerOnStart {
						con.triggerMessage(pending)
					}
					delete(con.pendingStreams, msg.receive.streamId)
				}
			} else {
				if !msg.isComplete {
					con.pendingStreams[msg.receive.streamId] = msg
					if triggerOnStart {
						con.triggerMessage(msg)
					}
				} else {
					con.triggerMessage(msg)
				}
			}
		}
	}
}
