package priority_test

import (
	"github.com/smartwalle/queue/priority"
	"math/rand"
	"testing"
	"time"
)

func BenchmarkPriorityQueue_Enqueue(b *testing.B) {
	var q = priority.New[int]()

	var m = make(map[int]int64)
	var r = rand.NewSource(time.Now().Unix())
	for i := 0; i < b.N; i++ {
		m[i] = r.Int63()
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		q.Enqueue(i, m[i])
	}
}
