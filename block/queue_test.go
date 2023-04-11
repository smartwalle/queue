package block_test

import (
	"github.com/smartwalle/queue/block"
	"sync"
	"testing"
)

func BenchmarkBlockQueue_Enqueue(b *testing.B) {
	var q = block.New[int]()

	for i := 0; i < b.N; i++ {
		q.Enqueue(i)
	}
}

func BenchmarkBlockQueue_EnqueueDequeue(b *testing.B) {
	var q = block.New[int]()
	var wg = &sync.WaitGroup{}
	go func() {
		var items []int
		for {
			items = items[0:0]
			var ok = q.Dequeue(&items)

			if len(items) > 0 {
				for range items {
					wg.Done()
				}
			}

			if !ok {
				break
			}
		}
	}()

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		q.Enqueue(i)
	}
	wg.Wait()
	q.Close()
}
