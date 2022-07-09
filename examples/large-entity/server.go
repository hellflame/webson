//go:build ignore
// +build ignore

package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/hellflame/webson"
)

func main() {
	http.HandleFunc("/large-entity", func(w http.ResponseWriter, r *http.Request) {
		ws, e := webson.TakeOver(w, r, &webson.Config{
			TriggerOnStart: true,
		})
		if e != nil {
			return
		}
		msgIdx := 0
		// monitor TextMessage process and read it at last
		ws.OnMessage(webson.TextMessage, func(m *webson.Message, a webson.Adapter) {
			// be careful, TriggerOnStart only means trigger on start, you need to:
			// loop read unfinished msg, until it's complete
			msgIdx += 1 // don't worry about synchronization problem
			for {
				msg, e := m.Read()
				if e != nil {
					switch e.(type) {
					case webson.MsgYetComplete:
						fmt.Println("waiting for completing msg....")
						time.Sleep(time.Second)
						continue
					default:
						panic(e)
					}
				}
				fmt.Println("here comes the complete msg")

				save, e := os.Create(fmt.Sprintf("%d.txt", msgIdx))
				if e != nil {
					panic(e)
				}
				save.Write(msg)
				save.Close()

				fmt.Println("TextMessage is saved")
				break // break out the loop after msg is complete
			}
		})

		// use ReadIter to process msg chunk by chunk
		ws.OnMessage(webson.BinaryMessage, func(m *webson.Message, a webson.Adapter) {
			save, e := os.Create("random.bin")
			if e != nil {
				panic(e)
			}
			defer save.Close()

			h := sha1.New()
			for chunk := range m.ReadIter(2) {
				save.Write(chunk)
				h.Write(chunk)
				a.Dispatch(webson.TextMessage, []byte("chunk saved")) // chunk ack
			}
			a.Dispatch(webson.TextMessage, []byte("random finished"))
			fmt.Println("received sha1", hex.EncodeToString(h.Sum(nil)))
		})

		// don't forget to Start
		if e := ws.Start(); e != nil {
			panic(e)
		}
	})
	fmt.Println("waiting for connections....")
	http.ListenAndServe("127.0.0.1:8000", nil)
}
