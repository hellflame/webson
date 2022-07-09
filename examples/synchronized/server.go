//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"net/http"

	"github.com/hellflame/webson"
)

func main() {
	http.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
		ws, e := webson.TakeOver(w, r, &webson.Config{Synchronize: true})
		if e != nil {
			return
		}
		// on status is still triggered Asynchronized
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
	if e := http.ListenAndServe("127.0.0.1:8000", nil); e != nil {
		panic(e)
	}
}
