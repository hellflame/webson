//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hellflame/webson"
)

type msg struct {
	Topic   string `json:"topic"`
	Content string `json:"content"`
}

func main() {
	port := "8000"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}
	ws, e := webson.Dial("ws://127.0.0.1:"+port+"/client", nil)
	if e != nil {
		panic(e)
	}

	loopQuest := func(a webson.Adapter) {
		var topic string
		fmt.Println("which topic you want to join?")
		fmt.Scanln(&topic)
		if topic == "" {
			fmt.Println("default topic is choosed")
			topic = "default"
		}
		fmt.Println("send anything to the topic?")
		var input string
		for {
			fmt.Scanln(&input)
			if input == "" {
				continue
			}
			js, _ := json.Marshal(msg{Topic: topic, Content: input})
			a.Dispatch(webson.TextMessage, js)
		}
	}

	ws.OnReady(func(a webson.Adapter) {
		loopQuest(a)
	})

	if e := ws.Start(); e != nil {
		panic(e)
	}
}
