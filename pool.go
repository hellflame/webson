package webson

import (
	"errors"
	"sync"
	"time"
)

type Pool struct {
	clients []*Connection
	servers []*Connection
	config  *PoolConfig

	entryMap map[string]*Connection

	poolLock sync.Mutex
	closed   bool
}

func NewPool(c *PoolConfig) *Pool {
	if c == nil {
		c = &PoolConfig{}
	}
	return &Pool{
		config: c,
	}
}

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

	connectionName := c.Name()
	if _, exist := p.entryMap[connectionName]; exist {
		return errors.New("connection name conflict")
	}
	if p.config.Size != 0 && len(p.entryMap) >= p.config.Size {
		return errors.New("pool size exceeded")
	}
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

func (p *Pool) remove(c *Connection, isClient bool) {
	name := c.Name()
	p.poolLock.Lock()
	defer p.poolLock.Unlock()
	delete(p.entryMap, name)
	idx := -1
	search := p.servers
	if isClient {
		search = p.clients
	}
	for i, c := range search {
		if c.Name() == name {
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
	p.remove(c, true)
}

func (p *Pool) startServer(c *Connection) {
	c.Start()
	p.remove(c, false)
}

func (p *Pool) CastOut(c *Connection) {

}

func (p *Pool) Dispatch(t MessageType, payload []byte) {

}

func (p *Pool) Clients() *Pool {
	return nil
}

func (p *Pool) Servers() *Pool {
	return nil
}

func (p *Pool) Group(name string) *Pool {
	return nil
}

func (p *Pool) Pick(name string) Adapter {
	return nil
}

func (p *Pool) Except(name string) *Pool {
	return nil
}

func (p *Pool) Close() {
	p.closed = true
	for _, c := range p.entryMap {
		c.Close()
	}
	p.Wait()
}

func (p *Pool) Wait() {
	for {
		time.Sleep(time.Duration(DEFAUTL_RETRY_INTERVAL) * time.Second)

		p.poolLock.Lock()
		isEmpty := len(p.entryMap) <= 0
		p.poolLock.Unlock()

		if isEmpty {
			break
		}
	}
}
