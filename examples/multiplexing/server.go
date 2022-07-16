//go:build ignore
// +build ignore

package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"

	"github.com/hellflame/webson"
)

func main() {
	http.HandleFunc("/multiplexing", func(w http.ResponseWriter, r *http.Request) {
		ws, e := webson.TakeOver(w, r, &webson.Config{
			EnableStreams:  true,
			TriggerOnStart: true,
		})
		if e != nil {
			panic(e)
		}
		totalMessages := 0
		lock := new(sync.Mutex)
		incr := func() {
			lock.Lock()
			totalMessages += 1
			lock.Unlock()
		}
		ws.OnStatus(webson.StatusClosed, func(s webson.Status, a webson.Adapter) {
			fmt.Println("connection is closed with total messages", totalMessages)
		})

		ws.OnMessage(webson.BinaryMessage, func(m *webson.Message, a webson.Adapter) {
			fmt.Println("binary received")
			h := sha1.New()
			for chunk := range m.ReadIter(2) {
				h.Write(chunk)
			}
			fmt.Println("finish", hex.EncodeToString(h.Sum(nil)))
			incr()
		})

		// don't forget to Start
		if e := ws.Start(); e != nil {
			panic(e)
		}
	})
	fmt.Println("waiting for connections....")
	http.ListenAndServe("127.0.0.1:8000", nil)
}
