package main

import (
	"fmt"

	"github.com/smartwalle/queue/priorityqueue"
)

// task 是带优先级的任务，priority 越小优先级越高。
type task struct {
	name     string
	priority int
}

func main() {
	// less 返回 true 表示 a 应比 b 先出队。
	var pQueue = priorityqueue.New[task](func(a, b task) bool {
		return a.priority < b.priority
	})

	for _, t := range []task{
		{"1", 1},
		{"2", 0},
		{"3", 5},
		{"4", 2},
		{"5", 9},
		{"6", 0},
	} {
		if err := pQueue.Enqueue(t); err != nil {
			fmt.Println("Enqueue", t.name, err)
		}
	}

	// Dequeue 在队列为空时会阻塞等待，先关闭再排空。
	pQueue.Close()

	for {
		var t, err = pQueue.Dequeue()
		if err != nil {
			break
		}
		fmt.Println("Dequeue", t.name, t.priority)
	}
}
