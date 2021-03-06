package webson

import (
	"errors"
	"sync"
	"time"
)

type poolEventProxy struct {
	name           string
	statusHandler  func(Status, Adapter)
	messageHandler func(*Message, Adapter)
}

func (p *poolEventProxy) Name() string {
	return p.name
}

func (p *poolEventProxy) OnStatus(s Status, a Adapter) {
	if p.statusHandler == nil {
		return
	}
	p.statusHandler(s, a)
}

func (p *poolEventProxy) OnMessage(m *Message, a Adapter) {
	if p.messageHandler == nil {
		return
	}
	p.messageHandler(m, a)
}

// Pool is the connection pool for any client or server connections.
// Don't create one just use &Pool{xxx}, use NewPool instead.
type Pool struct {
	clients []*Connection
	servers []*Connection
	config  *PoolConfig

	entryMap map[string]*Connection

	poolEventProxy

	poolLock sync.Mutex
	closed   bool
}

// NewPool create a usable connection pool
func NewPool(c *PoolConfig) *Pool {
	if c == nil {
		c = &PoolConfig{}
	}
	if c.Name == "" {
		c.Name = createChallengeKey()
	}
	return &Pool{
		config:   c,
		entryMap: make(map[string]*Connection),

		poolEventProxy: poolEventProxy{name: c.Name},
	}
}

// OnStatus will bind status handler for all connections
func (p *Pool) OnStatus(action func(Status, Adapter)) {
	p.statusHandler = action
}

// OnMessage will bind message handler for all connections
func (p *Pool) OnMessage(action func(*Message, Adapter)) {
	p.messageHandler = action
}

// Add takes one connection to the pool, it can be a client or server connection
func (p *Pool) Add(c *Connection, config *NodeConfig) error {
	if config == nil {
		config = &NodeConfig{}
	}
	if config.Name == "" {
		config.Name = createChallengeKey()
	}
	c.node = config

	p.poolLock.Lock()
	defer p.poolLock.Unlock()

	connectionName := c.node.Name
	if _, exist := p.entryMap[connectionName]; exist {
		return errors.New("connection name conflict")
	}
	if p.config.Size != 0 && len(p.entryMap) >= p.config.Size {
		return errors.New("pool size exceeded")
	}

	c.Apply(&p.poolEventProxy)
	p.entryMap[connectionName] = c

	if c.isClient {
		p.clients = append(p.clients, c)
		go p.startClient(c)
	} else {
		p.servers = append(p.servers, c)
		go p.startServer(c)
	}

	return nil
}

func (p *Pool) remove(c *Connection) {
	name := c.node.Name
	isClient := c.isClient

	p.poolLock.Lock()
	defer p.poolLock.Unlock()

	delete(p.entryMap, name)
	c.Revoke(p.name)

	idx := -1
	search := p.servers

	if isClient {
		search = p.clients
	}
	for i, c := range search {
		if c.node.Name == name {
			idx = i
			break
		}
	}
	if idx >= 0 {
		if isClient {
			p.clients = append(p.clients[:idx], p.clients[idx+1:]...)
		} else {
			p.servers = append(p.servers[:idx], p.servers[idx+1:]...)
		}
	}
}

func (p *Pool) startClient(c *Connection) {
	retry := p.config.ClientRetry
	for {
		c.Start()
		if retry > 0 && !p.closed {
			time.Sleep(time.Duration(p.config.RetryInterval) * time.Second)
			retry -= 1
		} else {
			break
		}
	}
	p.remove(c)
}

func (p *Pool) startServer(c *Connection) {
	c.Start()
	p.remove(c)
}

// CastOut a connection
func (p *Pool) CastOut(c *Connection) {
	p.remove(c)
}

// CastOut a connection with the given name
func (p *Pool) CastOutByName(name string) bool {
	p.poolLock.Lock()
	c, ok := p.entryMap[name]
	p.poolLock.Unlock()

	if ok {
		p.remove(c)
	}
	return ok
}

// Dispatch will broadcast the message to all connections in the pool
func (p *Pool) Dispatch(t MessageType, payload []byte) {
	p.poolLock.Lock()
	defer p.poolLock.Unlock()

	for _, c := range p.entryMap {
		c.Dispatch(t, payload)
	}
}

// ToClients will broadcast the message to client side connections (from Dial)
func (p *Pool) ToClients(t MessageType, payload []byte) {
	p.poolLock.Lock()
	defer p.poolLock.Unlock()
	for _, c := range p.clients {
		c.Dispatch(t, payload)
	}
}

// ToServers will broadcast the message to server side connections (from TakeOver)
func (p *Pool) ToServers(t MessageType, payload []byte) {
	p.poolLock.Lock()
	defer p.poolLock.Unlock()
	for _, c := range p.servers {
		c.Dispatch(t, payload)
	}
}

// ToClients will broadcast the message to the given group connections
func (p *Pool) ToGroup(gName string, t MessageType, payload []byte) {
	p.poolLock.Lock()
	defer p.poolLock.Unlock()

	for _, c := range p.entryMap {
		if c.node.Group == gName {
			c.Dispatch(t, payload)
		}
	}
}

// ToPick will try to send message to the connection with given name
func (p *Pool) ToPick(name string, t MessageType, payload []byte) bool {
	p.poolLock.Lock()
	defer p.poolLock.Unlock()

	if c, exist := p.entryMap[name]; exist {
		c.Dispatch(t, payload)
		return true
	}
	return false
}

// Except will broadcast message to all connections except the given name
func (p *Pool) Except(name string, t MessageType, payload []byte) {
	p.poolLock.Lock()
	defer p.poolLock.Unlock()

	for n, c := range p.entryMap {
		if n == name {
			continue
		}
		c.Dispatch(t, payload)
	}
}

// Close the pool, return when all connection closed
func (p *Pool) Close() {
	p.closed = true
	p.poolLock.Lock()
	for _, c := range p.entryMap {
		c.Close()
	}
	p.poolLock.Unlock()
	p.Wait()
}

// Wait will wait until all connection is dead.
func (p *Pool) Wait() {
	for {
		time.Sleep(time.Duration(client_retry_interval) * time.Second)

		p.poolLock.Lock()
		stop := len(p.entryMap) <= 0 && p.closed
		p.poolLock.Unlock()

		if stop {
			break
		}
	}
}
