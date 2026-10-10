package main

import (
	"fmt"
	"time"

	"github.com/smartwalle/queue/blockingqueue"
)

func main() {
	// max <= 0 时为无界队列。
	var bQueue = blockingqueue.New[string](0)

	var done = make(chan struct{})
	go func() {
		defer close(done)

		for {
			fmt.Println("read...")
			var item, err = bQueue.Dequeue()
			if err != nil {
				// 队列已关闭且排空。
				break
			}
			fmt.Println("Dequeue", item)
		}

		fmt.Println("stop....")
	}()

	time.Sleep(time.Second)
	bQueue.Enqueue("1")
	bQueue.Enqueue("2")

	time.Sleep(time.Second)
	bQueue.Enqueue("3")
	bQueue.Enqueue("4")

	time.Sleep(time.Second)
	// Close 只关闭队列，不等待存量元素出队。
	bQueue.Close()
	// Wait 等待存量元素全部出队。
	bQueue.Wait()

	<-done
}
