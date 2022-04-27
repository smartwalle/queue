package priority_test

import (
	"github.com/smartwalle/queue/priority"
	"testing"
)

func BenchmarkPriorityQueue_Enqueue(b *testing.B) {
	var q = priority.New[int]()

	for i := 0; i < b.N; i++ {
		q.Enqueue(i, int64(i))
	}

	b.Log(b.N)
}
