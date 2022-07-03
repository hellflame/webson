//go:build ignore
// +build ignore

package main

import (
	"fmt"

	"github.com/hellflame/webson"
)

func main() {
	ws, e := webson.Dial("127.0.0.1:8000", nil)
	if e != nil {
		panic(e)
	}
	ws.OnReady(func(a webson.Adapter) {
		a.Dispatch(webson.TextMessage, []byte("hello"))
	})
	ws.OnMessage(webson.TextMessage, func(m *webson.Message, a webson.Adapter) {
		msg, _ := m.Read()
		if string(msg) == "recv: hello" {
			fmt.Println("close after server hello")
			a.Close()
		} else {
			// you may receive this message
			fmt.Println("from server:", string(msg))
		}
	})

	if e := ws.Start(); e != nil {
		panic(e)
	}
}
