//go:build ignore
// +build ignore

package main

import (
	"fmt"

	"github.com/hellflame/webson"
)

func main() {
	ws, e := webson.Dial("ws://127.0.0.1:8000/propagate", nil)
	if e != nil {
		panic(e)
	}
	ws.OnReady(func(a webson.Adapter) {
		a.Dispatch(webson.TextMessage, []byte("ready"))
	})
	ws.OnMessage(webson.TextMessage, func(m *webson.Message, a webson.Adapter) {
		msg, _ := m.Read()
		fmt.Println("from server:", string(msg))
	})

	if e := ws.Start(); e != nil {
		panic(e)
	}
}
