//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"

	"github.com/hellflame/webson"
)

func main() {
	var topic string
	if len(os.Args) > 1 {
		topic = os.Args[1]
	}

	ws, e := webson.Dial("ws://127.0.0.1:8000/topics", &webson.DialConfig{
		ClientConfig: webson.ClientConfig{ExtraHeaders: map[string]string{
			"topic": topic,
		}},
	})
	if e != nil {
		panic(e)
	}
	fmt.Println("waiting for messages......")
	ws.OnMessage(webson.TextMessage, func(m *webson.Message, a webson.Adapter) {
		msg, _ := m.Read()
		fmt.Println("topic message:", string(msg))
	})

	if e := ws.Start(); e != nil {
		panic(e)
	}
}
