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
	ws.OnReady(collectUserInput)
	ws.OnMessage(webson.TextMessage, printTextResponse)

	if e := ws.Start(); e != nil {
		panic(e)
	}
}

func collectUserInput(a webson.Adapter) {
	var input string
	fmt.Println("input something and press ENTER")
	for {
		fmt.Scanln(&input)
		if input == "" {
			continue
		}
		a.Dispatch(webson.TextMessage, []byte(input))
	}
}

func printTextResponse(m *webson.Message, a webson.Adapter) {
	msg, _ := m.Read()
	fmt.Println("from server:", string(msg))
}
