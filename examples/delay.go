package main

import (
	"fmt"
	"github.com/smartwalle/queue/delay"
	"time"
)

func main() {
	var dQueue = delay.New(
		delay.WithTimeUnit(time.Second),
		delay.WithTimeProvider(func() int64 {
			return time.Now().Unix()
		}),
	)

	go func() {
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

	dQueue.Close()
	time.Sleep(time.Second * 2)
}
