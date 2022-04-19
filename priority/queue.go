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

// Queue 优先级队列
// 队列中元素的 priority 值越低，其优先级越高
type Queue interface {
	// Len 获取队列元素数量
	Len() int

	// Enqueue 添加元素到队列
	// 参数 priority 的值不能小于 0
	// 如果新元素在队列的头部（即为最高优先级元素），则返回 true，否则返回 false
	Enqueue(value interface{}, priority int64) bool

	// Dequeue 获取队列中的第一个元素及其优先级，并且将该元素从队列中删除
	// 如果队列中没有元素，则返回 nil 和 -1
	Dequeue() (interface{}, int64)

	// Peek 获取队列中优先级小于参数 max 的第一个元素，同时返回该元素的优先级和优先级与参数 max 的差值
	// 如果队列中没有元素，则返回值分别是 nil，-1 和 0
	// 如果队列中有元素，但是所有元素的优先级都大于参数 max 的值，则返回值分别是 nil，队列中第一个元素的优先级，队列中第一个元素的优先级与 max 的差值
	// 如果队列中有元素，并且有元素的优先级小于等于参数 max 的值，则返回值分别是队列中第一个元素，队列中第一个元素的优先级，0。并且将该元素从队列中删除
	Peek(max int64) (interface{}, int64, int64)
}

type priorityQueue []*Item

func New(capacity int) Queue {
	var nQueue = make(priorityQueue, 0, capacity)
	return &nQueue
}

func (pq priorityQueue) Len() int {
	return len(pq)
}

func (pq priorityQueue) Less(i, j int) bool {
	return pq[i].Priority < pq[j].Priority
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	c := cap(*pq)
	if n+1 > c {
		npq := make(priorityQueue, n, c*2)
		copy(npq, *pq)
		*pq = npq
	}
	*pq = (*pq)[0 : n+1]
	item := x.(*Item)
	item.Index = n
	(*pq)[n] = item
}

func (pq *priorityQueue) Pop() interface{} {
	n := len(*pq)
	c := cap(*pq)
	if n < (c/2) && c > 25 {
		npq := make(priorityQueue, n, c/2)
		copy(npq, *pq)
		*pq = npq
	}
	item := (*pq)[n-1]
	item.Index = -1
	*pq = (*pq)[0 : n-1]
	return item
}

func (pq *priorityQueue) Enqueue(value interface{}, priority int64) bool {
	if priority < 0 {
		priority = 0
	}
	var item = &Item{Value: value, Priority: priority}
	heap.Push(pq, item)
	return item.Index == 0
}

func (pq *priorityQueue) Dequeue() (interface{}, int64) {
	if pq.Len() == 0 {
		return nil, -1
	}
	item := heap.Pop(pq).(*Item)
	return item.Value, item.Priority
}

func (pq *priorityQueue) Peek(max int64) (interface{}, int64, int64) {
	if pq.Len() == 0 {
		return nil, -1, 0
	}

	item := (*pq)[0]
	if item.Priority > max {
		return nil, item.Priority, item.Priority - max
	}
	heap.Remove(pq, 0)

	return item.Value, item.Priority, 0
}
