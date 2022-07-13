//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"os"

	"github.com/hellflame/webson"
)

func main() {
	port := "8000"
	var topic string
	if len(os.Args) > 1 {
		topic = os.Args[1]
	}
	if len(os.Args) > 2 {
		port = os.Args[2]
	}

	ws, e := webson.Dial("ws://127.0.0.1:"+port+"/topics", &webson.DialConfig{
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
		fmt.Println("from server:", string(msg))
	})

	if e := ws.Start(); e != nil {
		panic(e)
	}
}
