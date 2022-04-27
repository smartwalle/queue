package block_test

import (
	"github.com/smartwalle/queue/block"
	"testing"
)

func BenchmarkBlockQueue_Enqueue(b *testing.B) {
	var q = block.New[int]()

	for i := 0; i < b.N; i++ {
		q.Enqueue(i)
	}

	b.Log(b.N)
}
