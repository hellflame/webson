package webson

import (
	"fmt"
	"net/http"
	"testing"
)

func TestService(t *testing.T) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ws, e := TakeOver(w, r, nil)
		if e != nil {
			t.Error(e)
			return
		}
		fmt.Printf("%+v", ws)
	})
	if e := http.ListenAndServe("127.0.0.1:8000", nil); e != nil {
		t.Error(e)
	}
}
