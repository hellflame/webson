//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hellflame/webson"
)

type msg struct {
	Topic   string `json:"topic"`
	Content string `json:"content"`
}

func main() {
	pool := webson.NewPool(nil)

	http.HandleFunc("/topics", func(w http.ResponseWriter, r *http.Request) {
		topicName := r.Header.Get("topic")
		if topicName == "" {
			topicName = "default"
		}
		fmt.Println("topic listener added for", topicName)
		ws, e := webson.TakeOver(w, r, nil)
		if e != nil {
			return
		}
		ws.OnReady(func(a webson.Adapter) {
			a.Dispatch(webson.TextMessage, []byte("ready"))
		})
		ws.OnStatus(webson.StatusClosed, func(s webson.Status, a webson.Adapter) {
			fmt.Println("one client left", topicName)
		})

		if e := pool.Add(ws, &webson.NodeConfig{Group: topicName}); e != nil {
			fmt.Println(e.Error())
		}
	})

	http.HandleFunc("/client", func(w http.ResponseWriter, r *http.Request) {
		ws, e := webson.TakeOver(w, r, nil)
		if e != nil {
			return
		}
		ws.OnMessage(webson.TextMessage, func(m *webson.Message, a webson.Adapter) {
			rawClientMsg, _ := m.Read()
			var clientMsg msg
			if json.Unmarshal(rawClientMsg, &clientMsg) != nil {
				fmt.Println("client msg format error")
				return
			}
			pool.ToGroup(clientMsg.Topic, webson.TextMessage, []byte(clientMsg.Content))
		})

		// don't forget to Start this non-grouped connection
		ws.Start()
	})

	fmt.Println("waiting for connections....")
	if e := http.ListenAndServe("127.0.0.1:8000", nil); e != nil {
		panic(e)
	}
}
