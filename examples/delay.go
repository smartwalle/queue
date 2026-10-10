package main

import (
	"fmt"
	"time"

	"github.com/smartwalle/queue/delayqueue"
)

// delayedMsg 是带到期时间的消息。
type delayedMsg struct {
	value    string
	deadline time.Time
}

func main() {
	var dQueue = delayqueue.New[delayedMsg](func(v delayedMsg) time.Time {
		return v.deadline
	})

	var done = make(chan struct{})
	go func() {
		defer close(done)
		for {
			var v, err = dQueue.Dequeue()
			if err != nil {
				break
			}
			fmt.Println("Dequeue", time.Now().Unix(), v.value)
		}
	}()

	go func() {
		var i = 1
		for {
			var now = time.Now()
			dQueue.Enqueue(delayedMsg{value: fmt.Sprintf("sss%d-3", i), deadline: now.Add(time.Second * 3)})
			dQueue.Enqueue(delayedMsg{value: fmt.Sprintf("sss%d-5", i), deadline: now.Add(time.Second * 5)})
			dQueue.Enqueue(delayedMsg{value: fmt.Sprintf("sss%d-1", i), deadline: now.Add(time.Second * 1)})
			time.Sleep(time.Second)
			i++
		}
	}()

	time.Sleep(time.Second * 5)
	dQueue.Enqueue(delayedMsg{value: "有效", deadline: time.Now()})
	time.Sleep(time.Second)

	// Close 关闭队列，不再接受新元素，已入队的元素到期后仍会取出。
	dQueue.Close()
	fmt.Println("Enqueue 无效 ->", dQueue.Enqueue(delayedMsg{value: "无效", deadline: time.Now()}))

	<-done
	fmt.Println("队列已排空")
}
