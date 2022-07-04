package webson

// Status stand for the state of the websocket.
type Status int

const (
	// StatusYetReady is the state before handshake confirm.
	// This is the first state of the connection.
	StatusYetReady = Status(-1)

	// StatusReady is the state when this side is ready to send and reveive message
	StatusReady = Status(0)

	// StatusClosed is the state when the connection is closed, normally or abnormally
	StatusClosed = Status(1)

	// StatusTimeout is the state when any action has spent more than expected time:
	// handshake, wait for close, pong, etc.
	// This state may be triggered for multiple times from recovery to timeout.
	StatusTimeout = Status(2)
)
