package priority

// from https://github.com/nsqio/nsq/blob/master/internal/pqueue/pqueue.go

import (
	"container/heap"
)

type Item struct {
	Value    interface{}
	Priority int64
	Index    int
}

type Queue []*Item

func New(capacity int) Queue {
	return make(Queue, 0, capacity)
}

func (pq Queue) Len() int {
	return len(pq)
}

func (pq Queue) Less(i, j int) bool {
	return pq[i].Priority < pq[j].Priority
}

func (pq Queue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *Queue) Push(x interface{}) {
	n := len(*pq)
	c := cap(*pq)
	if n+1 > c {
		npq := make(Queue, n, c*2)
		copy(npq, *pq)
		*pq = npq
	}
	*pq = (*pq)[0 : n+1]
	item := x.(*Item)
	item.Index = n
	(*pq)[n] = item
}

func (pq *Queue) Pop() interface{} {
	n := len(*pq)
	c := cap(*pq)
	if n < (c/2) && c > 25 {
		npq := make(Queue, n, c/2)
		copy(npq, *pq)
		*pq = npq
	}
	item := (*pq)[n-1]
	item.Index = -1
	*pq = (*pq)[0 : n-1]
	return item
}

func (pq *Queue) PeekAndShift(max int64) (*Item, int64) {
	if pq.Len() == 0 {
		return nil, 0
	}

	item := (*pq)[0]
	if item.Priority > max {
		return nil, item.Priority - max
	}
	heap.Remove(pq, 0)

	return item, 0
}
