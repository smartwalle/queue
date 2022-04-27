package delay_test

import (
	"github.com/smartwalle/queue/delay"
	"testing"
)

func BenchmarkDelayQueue_Enqueue(b *testing.B) {
	var q = delay.New[int]()

	for i := 0; i < b.N; i++ {
		q.Enqueue(i, int64(i))
	}

	b.Log(b.N)
}
