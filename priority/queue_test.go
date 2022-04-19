package priority_test

import (
	"github.com/smartwalle/queue/priority"
	"testing"
)

func BenchmarkPriorityQueue_Enqueue(b *testing.B) {
	var q = priority.New(32)

	for i := 0; i < b.N; i++ {
		q.Enqueue(i, int64(i))
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		q.Dequeue()
	}

	b.Log(b.N)
}
