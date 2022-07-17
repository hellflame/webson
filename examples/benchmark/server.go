//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"net/http"

	"github.com/hellflame/webson"
)

func main() {
	http.HandleFunc("/benchmark", func(w http.ResponseWriter, r *http.Request) {
		ws, e := webson.TakeOver(w, r, &webson.Config{PingInterval: -1})
		if e != nil {
			return
		}
		ws.OnMessage(webson.TextMessage, func(m *webson.Message, a webson.Adapter) {
			msg, _ := m.Read()
			a.Dispatch(webson.TextMessage, msg)
		})

		// don't forget to Start
		if e := ws.Start(); e != nil {
			fmt.Println(e.Error())
		}
	})
	fmt.Println("waiting for connections....")
	if e := http.ListenAndServe("127.0.0.1:8000", nil); e != nil {
		panic(e)
	}
}
