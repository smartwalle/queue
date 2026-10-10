package blockingqueue_test

import (
	"sync"
	"testing"

	"github.com/smartwalle/queue/blockingqueue"
)

func BenchmarkBlockQueue_Enqueue(b *testing.B) {
	var q = blockingqueue.New[int](0)

	for i := 0; i < b.N; i++ {
		q.Enqueue(i)
	}
}

func BenchmarkBlockQueue_EnqueueDequeue(b *testing.B) {
	var q = blockingqueue.New[int](0)
	var wg = &sync.WaitGroup{}
	go func() {
		for {
			var _, err = q.Dequeue()

			if err != nil {
				break
			}

			wg.Done()
		}
	}()

	for i := 0; i < b.N; i++ {
		wg.Add(1)
		q.Enqueue(i)
	}
	wg.Wait()
	q.Close()
}
