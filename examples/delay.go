package main

import (
	"fmt"
	"github.com/smartwalle/queue/delay"
	"time"
)

func main() {
	var dQueue = delay.New[string](
		delay.WithDrainAll(),
		delay.WithTimeUnit(time.Second),
		delay.WithTimeProvider(func() int64 {
			return time.Now().Unix()
		}),
	)

	var done = make(chan struct{})
	go func() {
		defer func() {
			close(done)
		}()
		for {
			var v, p = dQueue.Dequeue()
			fmt.Println("Dequeue", time.Now().Unix(), v, p)
			if p == -1 {
				break
			}
		}
	}()

	go func() {
		var i = 1
		for {
			var now = time.Now()
			dQueue.Enqueue(fmt.Sprintf("sss%d-3", i), now.Add(time.Second*3).Unix())
			dQueue.Enqueue(fmt.Sprintf("sss%d-5", i), now.Add(time.Second*5).Unix())
			dQueue.Enqueue(fmt.Sprintf("sss%d-1", i), now.Add(time.Second*1).Unix())
			time.Sleep(time.Second * 1)
			i++
		}

	}()

	time.Sleep(time.Second * 5)
	dQueue.Enqueue("有效", time.Now().Unix())
	time.Sleep(time.Second * 1)
	dQueue.Close()
	dQueue.Enqueue("无效", time.Now().Add(time.Second*1).Unix())

	select {
	case <-done:
	}
}
