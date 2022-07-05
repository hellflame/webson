package webson

import "fmt"

type MsgYetComplete struct{}

func (e MsgYetComplete) Error() string {
	return ""
}

type MsgTooLarge struct{}

func (e MsgTooLarge) Error() string { return "" }

type CantWriteYet struct {
	Status
}

func (e CantWriteYet) Error() string {
	return fmt.Sprintf("can't write at status(%d)", e.Status)
}

type WriteAfterClose struct{}

func (e WriteAfterClose) Error() string {
	return ""
}
