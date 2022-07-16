//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/hellflame/webson"
)

func main() {
	pool := webson.NewPool(nil)

	nodeIdx := 0
	var idxLock sync.Mutex
	getNodeName := func() string {
		idxLock.Lock()
		defer idxLock.Unlock()
		nodeIdx += 1
		return strconv.Itoa(nodeIdx)
	}

	http.HandleFunc("/propagate", func(w http.ResponseWriter, r *http.Request) {
		ws, e := webson.TakeOver(w, r, nil)
		if e != nil {
			return
		}
		ws.OnReady(func(a webson.Adapter) {
			a.Dispatch(webson.TextMessage, []byte("ok"))
		})
		ws.OnStatus(webson.StatusClosed, func(s webson.Status, a webson.Adapter) {
			fmt.Println("connection is closed")
		})
		ws.OnMessage(webson.TextMessage, func(m *webson.Message, a webson.Adapter) {
			msg, _ := m.Read()
			nodeName := a.Name()
			fmt.Printf("%s is broadcasting", nodeName)
			pool.Except(nodeName, webson.TextMessage, msg)
		})
		pool.Add(ws, &webson.NodeConfig{Name: getNodeName()})
	})
	fmt.Println("waiting for connections....")
	http.ListenAndServe("127.0.0.1:8000", nil)
}
