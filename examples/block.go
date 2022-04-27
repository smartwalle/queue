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
			var ok = bQueue.Dequeue(&items)

			for _, item := range items {
				fmt.Println("Dequeue", item)
			}

			if !ok {
				break ReadLoop
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
