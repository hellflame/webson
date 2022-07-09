//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"strconv"

	"github.com/hellflame/webson"
)

func main() {
	ws, e := webson.Dial("127.0.0.1:8000/sync", &webson.DialConfig{
		Config: webson.Config{Synchronize: true},
	})
	if e != nil {
		panic(e)
	}
	ws.OnReady(func(a webson.Adapter) {
		for i := 0; i <= 10; i++ {
			a.Dispatch(webson.TextMessage, []byte(strconv.Itoa(i)))
		}
		a.Close()
	})
	ws.OnMessage(webson.TextMessage, func(m *webson.Message, a webson.Adapter) {
		msg, _ := m.Read()
		fmt.Println(string(msg))
	})

	if e := ws.Start(); e != nil {
		panic(e)
	}
}
