//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"

	"github.com/hellflame/webson"
)

func sendRandomStream(index int, a webson.Adapter) {
	buffer := bytes.NewBuffer(nil)
	vessel := make([]byte, 1024)
	h := sha1.New()
	for i := 0; i < 40; i++ {
		rand.Read(vessel)
		h.Write(vessel)
		buffer.Write(vessel)
	}
	fmt.Println("send sha1", index, hex.EncodeToString(h.Sum(nil)))
	if e := a.DispatchReader(webson.BinaryMessage, buffer); e != nil {
		fmt.Println(e.Error())
	}
}

func main() {
	ws, e := webson.Dial("127.0.0.1:8000/multiplexing", &webson.DialConfig{
		Config: webson.Config{
			EnableStreams: true,
		},
	})
	if e != nil {
		panic(e)
	}

	ws.OnReady(func(a webson.Adapter) {
		for i := 0; i < 30; i++ {
			func(index int) {
				go sendRandomStream(index, a)
			}(i)
		}
	})

	if e := ws.Start(); e != nil {
		panic(e)
	}
}
