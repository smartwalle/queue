package priority

// from https://github.com/nsqio/nsq/blob/master/internal/pqueue/pqueue.go

import (
	"container/heap"
	"sync"
)

type Element interface {
	IsFirst() bool

	getIndex() int

	updatePriority(int64)
}

type queueElement struct {
	value    interface{}
	priority int64
	index    int
}

func (ele *queueElement) IsFirst() bool {
	return ele.index == 0
}

func (ele *queueElement) getIndex() int {
	return ele.index
}

func (ele *queueElement) updatePriority(priority int64) {
	ele.priority = priority
}

// Queue 优先级队列
// 队列中元素的 priority 值越低，其优先级越高
type Queue interface {
	// Len 获取队列元素数量
	Len() int

	// Enqueue 添加元素到队列
	// 参数 priority 的值不能小于 0
	Enqueue(value interface{}, priority int64) Element

	// Dequeue 获取队列中的第一个元素及其优先级，并且将该元素从队列中删除
	// 如果队列中没有元素，则返回 nil 和 -1
	Dequeue() (interface{}, int64)

	// Peek 获取队列中优先级小于参数 max 的第一个元素，同时返回该元素的优先级和优先级与参数 max 的差值
	// 如果队列中没有元素，则返回值分别是：nil，-1 和 0
	// 如果队列中有元素，但是所有元素的优先级都大于参数 max 的值，则返回值分别是：nil，队列中第一个元素的优先级，队列中第一个元素的优先级与 max 的差值
	// 如果队列中有元素，并且有元素的优先级小于等于参数 max 的值，则返回值分别是：队列中第一个元素，队列中第一个元素的优先级，0。并且将该元素从队列中删除
	Peek(max int64) (interface{}, int64, int64)

	// Update 更新元素的优先级
	Update(ele Element, priority int64)

	// Remove 从队列中删除元素
	Remove(ele Element)
}

type priorityQueue struct {
	elements []*queueElement
	pool     *sync.Pool
}

func New() Queue {
	var q = &priorityQueue{}
	q.elements = make([]*queueElement, 0, 32)
	q.pool = &sync.Pool{
		New: func() interface{} {
			return &queueElement{}
		},
	}
	return q
}

func (pq *priorityQueue) Len() int {
	return len(pq.elements)
}

func (pq *priorityQueue) Less(i, j int) bool {
	return pq.elements[i].priority < pq.elements[j].priority
}

func (pq *priorityQueue) Swap(i, j int) {
	pq.elements[i], pq.elements[j] = pq.elements[j], pq.elements[i]
	pq.elements[i].index = i
	pq.elements[j].index = j
}

func (pq *priorityQueue) Push(x interface{}) {
	n := len(pq.elements)
	c := cap(pq.elements)
	if n+1 > c {
		npq := make([]*queueElement, n, c*2)
		copy(npq, pq.elements)
		pq.elements = npq
	}
	pq.elements = pq.elements[0 : n+1]
	ele := x.(*queueElement)
	ele.index = n
	pq.elements[n] = ele
}

func (pq *priorityQueue) Pop() interface{} {
	n := len(pq.elements)
	c := cap(pq.elements)
	if n < (c/2) && c > 32 {
		npq := make([]*queueElement, n, c/2)
		copy(npq, pq.elements)
		pq.elements = npq
	}
	var ele = pq.elements[n-1]
	ele.index = -1
	pq.elements = pq.elements[0 : n-1]
	return ele
}

func (pq *priorityQueue) Enqueue(value interface{}, priority int64) Element {
	if priority < 0 {
		priority = 0
	}
	var ele = pq.pool.Get().(*queueElement)
	ele.value = value
	ele.priority = priority

	heap.Push(pq, ele)
	return ele
}

func (pq *priorityQueue) Dequeue() (interface{}, int64) {
	if pq.Len() == 0 {
		return nil, -1
	}
	var ele = heap.Pop(pq).(*queueElement)

	var value = ele.value
	var priority = ele.priority

	ele.value = nil
	ele.priority = -1
	pq.pool.Put(ele)

	return value, priority
}

func (pq *priorityQueue) Peek(max int64) (interface{}, int64, int64) {
	if pq.Len() == 0 {
		return nil, -1, 0
	}

	var ele = pq.elements[0]
	if ele.priority > max {
		return nil, ele.priority, ele.priority - max
	}
	heap.Remove(pq, 0)

	var value = ele.value
	var priority = ele.priority

	ele.value = nil
	ele.priority = -1
	pq.pool.Put(ele)

	return value, priority, 0
}

func (pq *priorityQueue) Update(ele Element, priority int64) {
	if ele == nil || ele.getIndex() < 0 {
		return
	}

	if pq.elements[ele.getIndex()] != ele {
		return
	}

	if priority < 0 {
		priority = 0
	}
	ele.updatePriority(priority)

	heap.Fix(pq, ele.getIndex())
}

func (pq *priorityQueue) Remove(ele Element) {
	if ele == nil || ele.getIndex() < 0 {
		return
	}

	if pq.elements[ele.getIndex()] != ele {
		return
	}

	heap.Remove(pq, ele.getIndex())
}
