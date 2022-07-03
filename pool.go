package webson

type Pool struct {
	clients []*Connection
	servers []*Connection
	config  *PoolConfig
}

func NewPool(c *PoolConfig) *Pool {
	return &Pool{}
}

func (p *Pool) Add(c *Connection, config *NodeConfig) {

}

func (p *Pool) CastOut(c *Connection) {

}

func (p *Pool) OnMessage(t MessageType, action func(m *Message, a Adapter)) {
}

func (p *Pool) OnStatus(s Status, action func(s Status, a Adapter)) {

}

func (p *Pool) Dispatch(m *Message) {

}

func (p *Pool) Group(name string) *Pool {
	return nil
}

func (p *Pool) Close() {

}
