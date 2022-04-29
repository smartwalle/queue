package priority_test

import (
	"github.com/smartwalle/queue/priority"
	"math/rand"
	"sort"
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

func BenchmarkPriorityQueue_Dequeue(b *testing.B) {
	var q = priority.New[int]()

	var r = rand.NewSource(time.Now().Unix())
	var max int64
	for i := 0; i < b.N; i++ {
		var p = r.Int63()
		if p > max {
			max = p
		}

		q.Enqueue(i, p)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		q.Dequeue()
	}
}

func BenchmarkPriorityQueue_Peek(b *testing.B) {
	var q = priority.New[int]()

	var r = rand.NewSource(time.Now().Unix())
	var max int64
	for i := 0; i < b.N; i++ {
		var p = r.Int63()
		if p > max {
			max = p
		}

		q.Enqueue(i, p)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		q.Peek(max)
	}
}

func BenchmarkPriorityQueue_Remove(b *testing.B) {
	var q = priority.New[int]()

	var r = rand.NewSource(time.Now().Unix())
	var elements = make([]priority.Element, b.N)
	for i := 0; i < b.N; i++ {
		var p = r.Int63()
		var ele = q.Enqueue(i, p)
		elements[i] = ele
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Remove(elements[i])
	}
}

func TestPriorityQueue_Dequeue2(t *testing.T) {
	var q = priority.New[int]()

	var list = []int{6, 1, 8, 2, 9, 3, 5, 4, 7, 0}

	for _, item := range list {
		q.Enqueue(item, int64(item))
	}

	var nList = make([]int, 0, len(list))

	for {
		var item, p = q.Dequeue()
		if p == -1 {
			break
		}

		nList = append(nList, item)
	}

	sort.Ints(list)

	for idx, item := range nList {
		if item != list[idx] {
			t.Fatal("出队顺序与预期不符", list[idx], item)
		}
	}
}
