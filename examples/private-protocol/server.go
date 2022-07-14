//go:build ignore
// +build ignore

package main

import (
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/hellflame/webson"
)

func main() {
	http.HandleFunc("/private", func(w http.ResponseWriter, r *http.Request) {
		ws, e := webson.TakeOver(w, r, &webson.Config{
			AlwaysMask:    true,
			EnableStreams: true,
			MagicKey:      []byte("masking here & there"),
			PrivateMask:   []byte("bytes length is times 4!"),
			HeaderVerify: func(h http.Header) bool {
				fmt.Println("start header verifying ...")
				if h.Get("Secret") != "secretkey" {
					fmt.Println("Not Secret!!")
					return false
				}
				if h.Get("Authorization") != fmt.Sprintf("Basic %s",
					base64.StdEncoding.EncodeToString([]byte("user:pass"))) {
					fmt.Println("User & Password Miss Match!!")
					return false
				}
				fmt.Println("header verify passed.")
				return true
			},
		})
		if e != nil {
			return
		}

		ws.OnStatus(webson.StatusClosed, func(s webson.Status, a webson.Adapter) {
			fmt.Println("connection is closed")
		})
		ws.OnMessage(webson.TextMessage, func(m *webson.Message, a webson.Adapter) {
			msg, _ := m.Read()
			fmt.Println("recv client msg:", string(msg))
			a.Dispatch(webson.TextMessage, append([]byte("recv: "), msg...))
		})

		// don't forget to Start
		ws.Start()
	})
	fmt.Println("waiting for connections....")
	http.ListenAndServe("127.0.0.1:8000", nil)
}
