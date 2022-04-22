package main

import (
	"fmt"
	"github.com/smartwalle/queue/block"
	"time"
)

func main() {
	var bQueue = block.New()

	go func() {
		var items []interface{}

	ReadLoop:
		for {
			fmt.Println("read...")
			items = items[0:0]
			bQueue.Dequeue(&items)

			for _, item := range items {
				if item == nil {
					break ReadLoop
					return
				}
				fmt.Println("Dequeue", item)
			}
		}

		fmt.Println("stop....")
	}()

	time.Sleep(time.Second)
	bQueue.Enqueue("1")
	bQueue.Enqueue("2")

	time.Sleep(time.Second)
	bQueue.Enqueue("3")
	bQueue.Enqueue("4")

	time.Sleep(time.Second * 1)
	bQueue.Close()

	time.Sleep(time.Second * 5)

}
