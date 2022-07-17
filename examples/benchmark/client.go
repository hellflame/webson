//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/hellflame/webson"
)

type sample struct {
	msgSent  int
	duration int64
	e        error
}

type summary struct {
	TotalCons int
	FailCons  int

	TotalMsgs int
	TotalSec  float64

	ConsPerSec float64
	MsgsPerSec float64
}

func newSummary(spendSecond float64, samples []*sample) *summary {
	failCons := 0
	successConn := 0
	totalMsgs := 0
	var totalDuration int64 = 0
	for _, s := range samples {
		if s.e != nil {
			failCons += 1
			fmt.Println(s.e.Error())
			continue
		}
		successConn += 1
		totalMsgs += s.msgSent
		totalDuration += s.duration
	}
	consRate := 0.0
	msgRate := 0.0
	durationSecond := float64(totalDuration) / 1000
	if durationSecond > 0 {
		consRate = float64(successConn) / durationSecond
		msgRate = float64(totalMsgs) / durationSecond
	}
	return &summary{
		TotalCons:  len(samples),
		FailCons:   failCons,
		TotalMsgs:  totalMsgs,
		ConsPerSec: consRate,
		MsgsPerSec: msgRate,

		TotalSec: spendSecond,
	}
}

func endurance(msgCount int, msg []byte) (s *sample) {
	start := time.Now().UnixMilli()
	ws, e := webson.Dial("127.0.0.1:8000/benchmark", &webson.DialConfig{
		Config: webson.Config{PingInterval: -1},
	})
	if e != nil {
		return &sample{0, 0, e}
	}
	ws.OnReady(func(a webson.Adapter) {
		for i := 1; i <= msgCount; i++ {
			if err := a.Dispatch(webson.TextMessage, msg); err != nil {
				s = &sample{i, 0, e}
				break
			}
		}
		a.Close()
	})
	if e := ws.Start(); e != nil {
		return &sample{0, 0, e}
	}
	return &sample{msgCount, time.Now().UnixMilli() - start, nil}
}

func benchmark(total, concurrent, msgRepeat int, sampleMsg []byte) *summary {
	start := time.Now().UnixMilli()
	pool := make(chan int, concurrent)
	donePool := make(chan int, concurrent)
	finishSig := make(chan int)
	var totalSpent float64 = 0.0

	go func() {
		cnt := 0
		for i := 0; i < concurrent; i++ {
			pool <- 0
			cnt += 1
		}
		for {
			<-donePool
			cnt += 1
			if cnt > total {
				break
			}
			pool <- 0
		}
		close(pool)
		close(finishSig)
	}()

	var result = []*sample{}
	rLock := new(sync.Mutex)
	saveSample := func(s *sample) {
		rLock.Lock()
		defer rLock.Unlock()
		result = append(result, s)
	}

	for range pool {
		go func() {
			saveSample(endurance(msgRepeat, sampleMsg))
			donePool <- 1
		}()
	}

	for {
		<-finishSig
		totalSpent = (float64(time.Now().UnixMilli()) - float64(start)) / 1000
		time.Sleep(5 * time.Second)
		break
	}
	return newSummary(totalSpent, result)
}

func main() {
	fmt.Printf("connectivity summary: %+v\n", benchmark(1000, 100, 0, nil))
	fmt.Printf("dispatch summary: %+v\n", benchmark(1000, 10, 10, []byte("hello")))
}
