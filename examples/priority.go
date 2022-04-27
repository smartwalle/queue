package main

import (
	"fmt"
	"github.com/smartwalle/queue/priority"
)

func main() {
	var pQueue = priority.New[string]()

	fmt.Println("1", pQueue.Enqueue("1", 1))
	fmt.Println("2", pQueue.Enqueue("2", 0))
	fmt.Println("3", pQueue.Enqueue("3", 5))
	fmt.Println("4", pQueue.Enqueue("4", 2))
	fmt.Println("5", pQueue.Enqueue("5", 9))
	fmt.Println("6", pQueue.Enqueue("6", 0))

	for {
		var v, p = pQueue.Dequeue()
		if p == -1 {
			break
		}
		fmt.Println("Dequeue", v, p)
	}

	//var delay = int64(0)
	//for {
	//	var v, p, _ = pQueue.Peek(delay)
	//	fmt.Println("Peek", v, p)
	//	if v == nil {
	//		if p > 0 {
	//			delay = p
	//			continue
	//		}
	//		break
	//	}
	//}
}
