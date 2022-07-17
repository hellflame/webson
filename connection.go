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
	// for connection manage
	Close()
	// for message writing
	Ping() error
	Pong() error
	Dispatch(MessageType, []byte) error
	DispatchReader(MessageType, io.Reader) error
	// for heartbeat monitor
	RefreshPongTime()
	KeepPing(int, int)
	// for pool manage
	Name() string
	Group() string
}

type EventHandler interface {
	Name() string // yes, name is necessary, it's ok to return ""
	OnStatus(Status, Adapter)
	OnMessage(*Message, Adapter)
}

type Connection struct {
	rawConnection net.Conn

	pendingStreams map[int]*Message
	inUseStreams   map[int]struct{}
	lastStream     int
	streamIdLock   sync.Mutex

	isClient   bool
	status     Status
	startClose bool
	closed     bool
	started    bool
	lastPing   time.Time
	lastPong   time.Time
	statusLock sync.Mutex
	writeLock  sync.Mutex

	config *Config
	client *ClientConfig
	node   *NodeConfig
	negoSet

	// event map as default action, can be replaced.
	statusEventMap  map[Status]func(Status, Adapter)
	messageEventMap map[MessageType]func(*Message, Adapter)
	eventPool       []EventHandler
}

func (con *Connection) prepare() {

	con.rawConnection.SetDeadline(time.Time{})

	con.status = StatusYetReady
	con.statusEventMap = make(map[Status]func(Status, Adapter))
	con.messageEventMap = make(map[MessageType]func(*Message, Adapter))

	// if not streamable, pendingStreams is for continue frames
	con.pendingStreams = make(map[int]*Message)
	con.inUseStreams = make(map[int]struct{})

	// bind default pong for ping
	con.OnMessage(PingMessage, func(m *Message, a Adapter) {
		a.Pong()
	})
	con.OnMessage(PongMessage, func(m *Message, a Adapter) {
		a.RefreshPongTime()
	})

	if con.config.PingInterval > 0 {
		go con.KeepPing(con.config.PingInterval, con.config.Timeout.PongTimeout)
	}
}

func (con *Connection) Group() string {
	if con.node == nil {
		return ""
	}
	return con.node.Group
}

func (con *Connection) Name() string {
	if con.node == nil {
		return ""
	}
	return con.node.Name
}

func (con *Connection) cleanClose() {
	con.closed = true
	con.rawConnection.Close()
	// clear pending received streams
	for _, m := range con.pendingStreams {
		if !m.isComplete && m.receive.poolReading {
			close(m.receive.msgPool)
		}
	}
}

func (con *Connection) Close() {
	con.Dispatch(CloseMessage, nil)
	con.statusLock.Lock()
	if !con.startClose {
		con.startClose = true
		go con.makeSureClose()
	}
	con.statusLock.Unlock()
	con.updateStatus(StatusClosed)
}

func (con *Connection) makeSureClose() {
	time.Sleep(time.Duration(con.config.Timeout.CloseTimeout) * time.Second)
	if con.closed {
		return
	}
	con.cleanClose()
}

func (con *Connection) ReStart() error {
	// reset config
	negotiate(con.client, con.config)
	return con.Start()
}

func (con *Connection) Ping() error {
	con.lastPing = time.Now()
	return con.Dispatch(PingMessage, nil)
}

func (con *Connection) Pong() error {
	return con.Dispatch(PongMessage, nil)
}

func (con *Connection) RefreshPongTime() {
	con.lastPong = time.Now()
}

func (con *Connection) KeepPing(pingInterval, pongTimeout int) {
	timeout := false
	// pongTimeout := int64(con.config.Timeout.PongTimeout)
	for {
		switch con.Ping().(type) {
		case WriteAfterClose:
			break
		default:
		}
		time.Sleep(time.Duration(pingInterval) * time.Second)

		if pongTimeout > 0 {
			// check whether last ping has pong response
			if con.lastPong.Unix()-con.lastPing.Unix() > int64(pongTimeout) {
				if con.status == StatusClosed {
					break
				}
				timeout = true
				con.updateStatus(StatusTimeout)
			} else {
				if timeout {
					// remember to recover from timeout
					con.updateStatus(StatusReady)
				}
			}
		}
	}
}

func (con *Connection) Dispatch(t MessageType, p []byte) error {
	m := &Message{Type: t, payload: p}
	if e := con.patchMsg(m); e != nil {
		return e
	}
	if !con.streamable {
		con.writeLock.Lock()
		defer con.writeLock.Unlock()
	}

	for _, msg := range m.split(con.config.ChunkSize) {
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
		// only lock when it's not streaming, or dead lock may occur
		con.writeLock.Lock()
		defer con.writeLock.Unlock()
	}
	chunkSize := con.config.ChunkSize
	var vessel = make([]byte, chunkSize)
	for {
		n, e := r.Read(vessel)
		stream := msg.spawnVessel()
		stream.payload = vessel[0:n]

		if e != nil {
			if e == io.EOF {
				stream.isComplete = true
				return con.writeSingleFrame(stream)
			}
			stream.send.cancelStream = true
			if ew := con.writeSingleFrame(stream); ew != nil {
				return ew
			}
			return e
		}

		if e = con.writeSingleFrame(stream); e != nil {
			return e
		}
	}
}

// patchMsg keeps write Message intact & correct
func (con *Connection) patchMsg(m *Message) error {
	m.send = &msgSendOptions{
		doCompress:    con.compressable && !m.IsControl(),
		compressLevel: con.compressLevel,
		doMask:        con.isClient,
		streamlize:    con.streamable && !m.IsControl(),
	}
	m.config = &msgConfig{
		negotiate:      &con.negoSet,
		extraMask:      con.config.PrivateMask,
		triggerOnStart: con.config.TriggerOnStart,
	}

	if m.send.streamlize {
		con.streamIdLock.Lock()
		loop := false
		for {
			con.lastStream += 1
			if con.lastStream > con.config.MaxStreams {
				con.lastStream = 1
				loop = true
			}
			if _, inUse := con.inUseStreams[con.lastStream]; inUse {
				if loop {
					return errors.New("stream ids spent")
				}
			} else {
				con.inUseStreams[con.lastStream] = struct{}{}
				break
			}
		}

		m.send.streamId = con.lastStream
		con.streamIdLock.Unlock()
	}
	return nil
}

func (con *Connection) writeSingleFrame(m *Message) error {
	status := con.status
	if status == StatusClosed {
		return WriteAfterClose{}
	}
	if status != StatusReady {
		return CantWriteYet{status}
	}
	if e := m.assemble(); e != nil {
		return e
	}
	if m.send.streamlize {
		// only lock when it's streaming, or dead lock may occur
		con.writeLock.Lock()
		defer con.writeLock.Unlock()
		if m.isComplete {
			con.streamIdLock.Lock()
			delete(con.inUseStreams, m.send.streamId)
			con.streamIdLock.Unlock()
		}
	}
	_, e := io.Copy(con.rawConnection, &m.entity)
	return e
}

func (con *Connection) Apply(h EventHandler) {
	con.eventPool = append(con.eventPool, h)
}

func (con *Connection) Revoke(name string) {
	idx := -1
	for i, e := range con.eventPool {
		if e.Name() == name {
			idx = i
			break
		}
	}
	if idx >= 0 {
		con.eventPool = append(con.eventPool[:idx], con.eventPool[idx+1:]...)
	}
}

func (con *Connection) OnReady(action func(Adapter)) {
	con.OnStatus(StatusReady, func(s Status, a Adapter) {
		if s != StatusYetReady {
			// OnReady only handle status change from StatusYetReady
			return
		}
		action(a)
	})
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
		// mainly to prevent closed & timeout repeatly triggers
		return
	}
	con.status = s
	if action, ok := con.statusEventMap[s]; ok {
		go action(prevStatus, con)
	}

	for _, handler := range con.eventPool {
		go handler.OnStatus(s, con)
	}
	return
}

func (con *Connection) triggerMessage(m *Message) {
	if action, ok := con.messageEventMap[m.Type]; ok {
		if con.config.Synchronize {
			action(m, con)
		} else {
			go action(m, con)
		}
	}
	for _, handler := range con.eventPool {
		if con.config.Synchronize {
			handler.OnMessage(m, con)
		} else {
			go handler.OnMessage(m, con)
		}
	}
}

func (con *Connection) Start() error {
	defer func() {
		con.updateStatus(StatusClosed)
		con.cleanClose()
	}()

	reader := bufio.NewReaderSize(con.rawConnection, con.config.BufferSize)
	con.updateStatus(StatusReady)

	triggerOnStart := con.config.TriggerOnStart
	var vessel2 = make([]byte, 2)
	var vessel4 = make([]byte, 4)
	var vessel8 = make([]byte, 8)
	for {
		if s, e := reader.Read(vessel2); e != nil || s != 2 {
			return exceptEOF(e)
		}
		msg := &Message{
			config: &msgConfig{
				negotiate:      &con.negoSet,
				extraMask:      con.config.PrivateMask,
				triggerOnStart: triggerOnStart,
				synchronized:   con.config.Synchronize,
			},
			receive: &msgReceivedStatus{
				CreatedAt:    time.Now(),
				isFromClient: !con.isClient,
			},
		}
		if e := msg.parseMeta(vessel2); e != nil {
			return e
		}
		if msg.receive.size == 126 {
			if s, e := reader.Read(vessel2); e != nil || s != 2 {
				return errors.New("msg size not given")
			}
			msg.receive.size = int64(binary.BigEndian.Uint16(vessel2))
		} else if msg.receive.size == 127 {
			if s, e := reader.Read(vessel8); e != nil || s != 8 {
				return errors.New("msg size not given")
			}
			msg.receive.size = int64(binary.BigEndian.Uint64(vessel8))
		}

		if msg.receive.masked {
			if s, e := reader.Read(vessel4); e != nil || s != 4 {
				return errors.New("mask key not given")
			}
			msg.setMask(vessel4)
		}

		if msg.receive.size > 0 {
			if con.config.MaxPayloadSize != 0 && msg.receive.size > int64(con.config.MaxPayloadSize) {
				return MsgTooLarge{}
			}
			payload, e := reader.Peek(int(msg.receive.size))
			if e != nil {
				return exceptEOF(e)
			}
			reader.Discard(len(payload))
			if msg.receive.masked {
				msg.maskPayload(payload)
			}
			if msg.receive.isStream {
				cancel := payload[0]&0b1000_0000 != 0
				payload[0] = payload[0] & 0b0111_1111
				msg.receive.streamId = int(binary.BigEndian.Uint16(payload[:streamBytes]))
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

		if msg.IsControl() {
			con.triggerMessage(msg)
			if msg.Type == CloseMessage {
				// close ack message will be sent in defer function
				return nil
			}
		} else {
			// no matter streaming or not
			streamId := msg.receive.streamId
			if pending, exist := con.pendingStreams[streamId]; exist {
				// try to complete msg
				if e := pending.merge(msg); e != nil {
					return e
				}
				if msg.isComplete {
					if !triggerOnStart {
						con.triggerMessage(pending)
					}
					delete(con.pendingStreams, streamId)
				}
			} else {
				if !msg.isComplete {
					con.pendingStreams[streamId] = msg
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
