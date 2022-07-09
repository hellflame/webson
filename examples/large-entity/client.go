//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/hellflame/webson"
)

func main() {
	ws, e := webson.Dial("127.0.0.1:8000/large-entity", nil)
	if e != nil {
		panic(e)
	}

	ws.OnMessage(webson.TextMessage, func(m *webson.Message, a webson.Adapter) {
		msg, _ := m.Read()
		fmt.Println("server bin ack:", string(msg))
		if string(msg) == "random finished" {
			ws.Close()
		}
	})

	ws.OnReady(func(a webson.Adapter) {
		// send large text msg
		a.Dispatch(webson.TextMessage, []byte(strings.Repeat("hello\n", 1024)))
		a.Dispatch(webson.TextMessage, []byte(strings.Repeat("hello again\n", 1024)))

		buffer := bytes.NewBuffer(nil)
		vessel := make([]byte, 1024)
		h := sha1.New()
		for i := 0; i < 6; i++ {
			rand.Read(vessel)
			h.Write(vessel)
			buffer.Write(vessel)
		}
		fmt.Println("send sha1", hex.EncodeToString(h.Sum(nil)))
		a.DispatchReader(webson.BinaryMessage, buffer)
		a.Close()
	})

	if e := ws.Start(); e != nil {
		panic(e)
	}
}
