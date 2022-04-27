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

type queueElement[T any] struct {
	value    T
	priority int64
	index    int
}

func (ele *queueElement[T]) IsFirst() bool {
	return ele.index == 0
}

func (ele *queueElement[T]) getIndex() int {
	return ele.index
}

func (ele *queueElement[T]) updatePriority(priority int64) {
	ele.priority = priority
}

// Queue 优先级队列
// 队列中元素的 priority 值越低，其优先级越高
type Queue[T any] interface {
	// Len 获取队列元素数量
	Len() int

	// Enqueue 添加元素到队列
	// 参数 priority 的值不能小于 0
	Enqueue(value T, priority int64) Element

	// Dequeue 获取队列中的第一个元素及其优先级，并且将该元素从队列中删除
	// 如果队列中没有元素，则返回 nil 和 -1
	Dequeue() (T, int64)

	// Peek 获取队列中优先级小于参数 max 的第一个元素，同时返回该元素的优先级和优先级与参数 max 的差值，最后一个返回值为是否出队成功
	// 如果队列中没有元素，则返回值分别是：nil，-1，0 和 false
	// 如果队列中有元素，但是所有元素的优先级都大于参数 max 的值，则返回值分别是：nil，队列中第一个元素的优先级，队列中第一个元素的优先级与 max 的差值，false
	// 如果队列中有元素，并且有元素的优先级小于等于参数 max 的值，则返回值分别是：队列中第一个元素，队列中第一个元素的优先级，0。并且将该元素从队列中删除，true
	Peek(max int64) (T, int64, int64, bool)

	// Update 更新元素的优先级
	Update(ele Element, priority int64)

	// Remove 从队列中删除元素
	Remove(ele Element)
}

type priorityQueue[T any] struct {
	elements []*queueElement[T]
	pool     *sync.Pool
	empty    T
}

func New[T any]() Queue[T] {
	var q = &priorityQueue[T]{}
	q.elements = make([]*queueElement[T], 0, 32)
	q.pool = &sync.Pool{
		New: func() interface{} {
			return &queueElement[T]{}
		},
	}
	return q
}

func (pq *priorityQueue[T]) Len() int {
	return len(pq.elements)
}

func (pq *priorityQueue[T]) Less(i, j int) bool {
	return pq.elements[i].priority < pq.elements[j].priority
}

func (pq *priorityQueue[T]) Swap(i, j int) {
	pq.elements[i], pq.elements[j] = pq.elements[j], pq.elements[i]
	pq.elements[i].index = i
	pq.elements[j].index = j
}

func (pq *priorityQueue[T]) Push(x interface{}) {
	n := len(pq.elements)
	c := cap(pq.elements)
	if n+1 > c {
		npq := make([]*queueElement[T], n, c*2)
		copy(npq, pq.elements)
		pq.elements = npq
	}
	pq.elements = pq.elements[0 : n+1]
	ele := x.(*queueElement[T])
	ele.index = n
	pq.elements[n] = ele
}

func (pq *priorityQueue[T]) Pop() interface{} {
	n := len(pq.elements)
	c := cap(pq.elements)
	if n < (c/2) && c > 32 {
		npq := make([]*queueElement[T], n, c/2)
		copy(npq, pq.elements)
		pq.elements = npq
	}
	var ele = pq.elements[n-1]
	ele.index = -1
	pq.elements = pq.elements[0 : n-1]
	return ele
}

func (pq *priorityQueue[T]) Enqueue(value T, priority int64) Element {
	if priority < 0 {
		priority = 0
	}
	var ele = pq.pool.Get().(*queueElement[T])
	ele.value = value
	ele.priority = priority

	heap.Push(pq, ele)
	return ele
}

func (pq *priorityQueue[T]) Dequeue() (T, int64) {
	var value T
	if pq.Len() == 0 {
		return value, -1
	}
	var ele = heap.Pop(pq).(*queueElement[T])

	value = ele.value
	var priority = ele.priority

	ele.value = pq.empty
	ele.priority = -1
	ele.index = -1
	pq.pool.Put(ele)

	return value, priority
}

func (pq *priorityQueue[T]) Peek(max int64) (T, int64, int64, bool) {
	var value T
	if pq.Len() == 0 {
		return value, -1, 0, false
	}

	var ele = pq.elements[0]
	if ele.priority > max {
		return value, ele.priority, ele.priority - max, false
	}
	heap.Remove(pq, 0)

	value = ele.value
	var priority = ele.priority

	ele.value = pq.empty
	ele.priority = -1
	ele.index = -1
	pq.pool.Put(ele)

	return value, priority, 0, true
}

func (pq *priorityQueue[T]) Update(ele Element, priority int64) {
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

func (pq *priorityQueue[T]) Remove(ele Element) {
	if ele == nil || ele.getIndex() < 0 {
		return
	}

	if pq.elements[ele.getIndex()] != ele {
		return
	}

	heap.Remove(pq, ele.getIndex())
}
