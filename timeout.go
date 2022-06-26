package webson

type Timeout struct {
	HandshakeTimeout    int
	PongTimeout         int
	CloseConfirmTimeout int
	StreamWaitTimeout   int
}

func (t *Timeout) ResetDefaultTimeout() {
	t.SetAllTimeout(DEFAULT_TIMEOUT)
}

func (t *Timeout) SetAllTimeout(wait int) {
	t.HandshakeTimeout = wait
	t.PongTimeout = wait
	t.CloseConfirmTimeout = wait
	t.StreamWaitTimeout = wait
}
