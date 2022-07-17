//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"time"

	"github.com/hellflame/webson"
)

func main() {
	ws, e := webson.Dial("127.0.0.1:8000/override", &webson.DialConfig{
		Config: webson.Config{PingInterval: -1},
	})
	if e != nil {
		panic(e)
	}
	var lastPong int64 = 0
	updatePong := func() {
		fmt.Println("update last pong")
		lastPong = time.Now().Unix()
	}
	closedSig := make(chan int, 0)
	isTimeout := func() bool {
		return time.Now().Unix()-lastPong > 2
	}

	ws.OnReady(func(a webson.Adapter) {
		// Send Ping at every second, with no default timeout check
		// In the example, Only this side send ping
		go ws.KeepPing(1, 0)
		for {
			select {
			case <-closedSig:
				fmt.Println("closed")
				break
			}
			if isTimeout() {
				fmt.Println("time is out")
			}
			time.Sleep(time.Second)
		}
	})
	ws.OnStatus(webson.StatusClosed, func(s webson.Status, a webson.Adapter) {
		close(closedSig)
	})

	ws.OnMessage(webson.PingMessage, func(m *webson.Message, a webson.Adapter) {
		// Server is not disabled from sending Ping, this won't happen
		fmt.Println("ping should never recved")
		//a.Pong()
	})
	ws.OnMessage(webson.BinaryMessage, func(m *webson.Message, a webson.Adapter) {
		updatePong()
	})

	if e := ws.Start(); e != nil {
		panic(e)
	}
}
