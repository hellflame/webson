package webson

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
	return ""
}

type CantWrite struct{}

func (e CantWrite) Error() string {
	return ""
}
