//go:build ignore
// +build ignore

package main

import (
	"fmt"

	"github.com/hellflame/webson"
)

func main() {
	ws, e := webson.Dial("ws://user:pass@127.0.0.1:8000", &webson.DialConfig{
		Config: webson.Config{
			AlwaysMask:    true,
			EnableStreams: true,
			MagicKey:      []byte("masking here & there"),
			PrivateMask:   []byte("bytes length is times 4!"),
		},
		ClientConfig: webson.ClientConfig{
			ExtraHeaders: map[string]string{"Secret": "secretkey"},
		},
	})
	if e != nil {
		panic(e)
	}
	ws.OnReady(func(a webson.Adapter) {
		a.Dispatch(webson.TextMessage, []byte("hellflame is testing this connection!"))
		a.Dispatch(webson.TextMessage, []byte("hello"))
	})
	ws.OnMessage(webson.TextMessage, func(m *webson.Message, a webson.Adapter) {
		msgRaw, _ := m.Read()
		msg := string(msgRaw)
		fmt.Println("from server:", msg)
		if msg == "recv: hello" {
			fmt.Println("close after server hello")
			a.Close()
		}
	})

	if e := ws.Start(); e != nil {
		panic(e)
	}
}
