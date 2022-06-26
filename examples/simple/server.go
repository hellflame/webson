//go:build ignore
// +build ignore

package main

import (
	"net/http"

	"github.com/hellflame/webson"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		webson.TakeOver(w, r, nil)
	})
	http.ListenAndServe("127.0.0.1:8000", nil)
}
