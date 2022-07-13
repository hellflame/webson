//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/hellflame/webson"
)

type msg struct {
	Topic   string `json:"topic"`
	Content string `json:"content"`
}

func main() {
	topicPool := webson.NewPool(nil)

	nodePool := webson.NewPool(nil)
	port := "8000"
	if len(os.Args) > 1 && os.Args[1] != "8000" {
		// other node
		// be sure to user other port than 8000
		port = os.Args[1]

		// connect to the StartNode
		ws, e := webson.Dial("ws://127.0.0.1:8000/node", nil)
		if e != nil {
			panic(e)
		}

		// other node will take BinaryMessage as broadcast messages.
		ws.OnMessage(webson.BinaryMessage, func(m *webson.Message, a webson.Adapter) {
			rawClientMsg, _ := m.Read()
			var clientMsg msg
			if json.Unmarshal(rawClientMsg, &clientMsg) != nil {
				return
			}
			// broadcast messages to the local topic group.
			topicPool.ToGroup(clientMsg.Topic, webson.TextMessage, []byte(clientMsg.Content))
		})
		ws.OnReady(func(a webson.Adapter) {
			fmt.Println("connected to the StartNode")
		})
		ws.OnStatus(webson.StatusClosed, func(s webson.Status, a webson.Adapter) {
			fmt.Println("StartNode has disconnected")
		})

		nodePool.Add(ws, nil)
	}

	http.HandleFunc("/node", func(w http.ResponseWriter, r *http.Request) {
		ws, e := webson.TakeOver(w, r, nil)
		if e != nil {
			return
		}
		ws.OnReady(func(a webson.Adapter) {
			fmt.Println("one node has joined")
		})
		ws.OnStatus(webson.StatusClosed, func(s webson.Status, a webson.Adapter) {
			fmt.Println("one node has left")
		})
		if e := nodePool.Add(ws, nil); e != nil {
			fmt.Println(e.Error())
		}
	})

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

		if e := topicPool.Add(ws, &webson.NodeConfig{Group: topicName}); e != nil {
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
			topicPool.ToGroup(clientMsg.Topic, webson.TextMessage, []byte(clientMsg.Content))
			nodePool.Dispatch(webson.BinaryMessage, rawClientMsg) // send msg to other node
		})

		// don't forget to Start this non-grouped connection
		ws.Start()
	})

	fmt.Println("waiting for connections....")
	if e := http.ListenAndServe("127.0.0.1:"+port, nil); e != nil {
		panic(e)
	}
}
