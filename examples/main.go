package main

import (
	"fmt"
	"github.com/smartwalle/queue/delay"
	"time"
)

func main() {
	var dQueue = delay.New(
		delay.WithTimeUnit(time.Millisecond),
		delay.WithTimeProvider(func() int64 {
			return time.Now().UnixMilli()
		}),
	)

	go func() {
		for {
			var item = dQueue.Pop()
			fmt.Println("1", item)
		}
	}()

	go func() {
		var i = 0
		for {
			fmt.Println("==", i)
			dQueue.Push(fmt.Sprintf("sss3-%d", i), time.Now().Add(time.Second*3).UnixMilli())
			dQueue.Push(fmt.Sprintf("sss5-%d", i), time.Now().Add(time.Second*5).UnixMilli())
			dQueue.Push(fmt.Sprintf("sss1-%d", i), time.Now().Add(time.Second*1).UnixMilli())
			fmt.Println("==", i)
			time.Sleep(time.Second * 1)
			i++
		}
	}()

	select {}
}
