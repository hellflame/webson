//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"net/http"

	"github.com/hellflame/webson"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ws, e := webson.TakeOver(w, r, nil)
		if e != nil {
			return
		}
		ws.OnReady(func(a webson.Adapter) {
			a.Dispatch(webson.TextMessage, []byte("hellflame"))
		})
		ws.OnStatus(webson.StatusClosed, func(s webson.Status, a webson.Adapter) {
			fmt.Println("connection is closed")
		})
		ws.OnMessage(webson.TextMessage, func(m *webson.Message, a webson.Adapter) {
			msg, _ := m.Read()
			a.Dispatch(webson.TextMessage, append([]byte("recv: "), msg...))
		})

		// don't forget to Start
		ws.Start()
	})
	fmt.Println("waiting for connections....")
	http.ListenAndServe("127.0.0.1:8000", nil)
}
