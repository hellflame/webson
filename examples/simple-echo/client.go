//go:build ignore
// +build ignore

package main

import (
	"fmt"

	"github.com/hellflame/webson"
)

func main() {
	ws, e := webson.Dial("127.0.0.1:8000/echo", nil)
	if e != nil {
		panic(e)
	}

	fmt.Println("input something and press ENTER")
	ws.OnMessage(webson.TextMessage, collectAndEcho)

	if e := ws.Start(); e != nil {
		panic(e)
	}
}

func collectAndEcho(m *webson.Message, a webson.Adapter) {
	msg, _ := m.Read()
	fmt.Println("from server:", string(msg))

	var input string
	fmt.Scanln(&input)
	if input == "" {
		return
	}
	a.Dispatch(webson.TextMessage, []byte(input))
}
