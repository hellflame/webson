//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"net/http"

	"github.com/hellflame/webson"
)

func main() {
	http.HandleFunc("/override", func(w http.ResponseWriter, r *http.Request) {
		ws, e := webson.TakeOver(w, r, &webson.Config{
			PingInterval: -1,
		})
		if e != nil {
			return
		}

		ws.OnStatus(webson.StatusClosed, func(s webson.Status, a webson.Adapter) {
			fmt.Println("connection is closed")
		})

		ws.OnMessage(webson.PingMessage, func(m *webson.Message, a webson.Adapter) {
			// send a binary instead of Pong
			//a.Pong()
			fmt.Println("ping recved")
			if e := a.Dispatch(webson.BinaryMessage, nil); e != nil {
				fmt.Println(e.Error())
			}
		})

		// don't forget to Start
		ws.Start()
	})
	fmt.Println("waiting for connections....")
	http.ListenAndServe("127.0.0.1:8000", nil)
}
