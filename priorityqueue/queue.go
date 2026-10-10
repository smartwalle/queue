package priorityqueue

import (
	"container/heap"
	"sync"

	"github.com/smartwalle/queue"
)

type priorityHeap[T any] struct {
	elements []T
	less     func(T, T) bool
}

func (h *priorityHeap[T]) Len() int {
	return len(h.elements)
}

func (h *priorityHeap[T]) Less(i, j int) bool {
	return h.less(h.elements[i], h.elements[j])
}

func (h *priorityHeap[T]) Swap(i, j int) {
	h.elements[i], h.elements[j] = h.elements[j], h.elements[i]
}

func (h *priorityHeap[T]) Push(value any) {
	h.elements = append(h.elements, value.(T))
}

func (h *priorityHeap[T]) Pop() any {
	old := h.elements
	n := len(old)
	value := old[n-1]

	var zero T
	old[n-1] = zero
	h.elements = old[:n-1]

	return value
}

// Queue 优先级队列。
type Queue[T any] struct {
	mu   sync.Mutex
	cond *sync.Cond

	elements *priorityHeap[T]
	closed   bool
}

// New 创建优先级队列。
// less 用于比较元素的优先级，返回 true 表示第一个元素应优先出队。
func New[T any](less func(T, T) bool) *Queue[T] {
	var q = &Queue[T]{
		elements: &priorityHeap[T]{
			elements: make([]T, 0, 32),
			less:     less,
		},
	}

	q.cond = sync.NewCond(&q.mu)

	return q
}

// Len 获取队列当前的元素数量。
func (q *Queue[T]) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.elements.Len()
}

// Enqueue 添加元素到队列。
func (q *Queue[T]) Enqueue(value T) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return queue.ErrClosed
	}

	heap.Push(q.elements, value)
	q.cond.Signal()

	return nil
}

// TryDequeue 尝试获取优先级最高的元素，不会阻塞。
// 队列为空且已关闭时返回 queue.ErrClosed。
// 队列为空且未关闭时返回 queue.ErrEmpty。
func (q *Queue[T]) TryDequeue() (T, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.tryDequeue()
}

// Dequeue 获取优先级最高的元素。
// 队列为空时阻塞等待。
// 队列关闭后仍然可以取出剩余元素。
// 队列关闭且为空时返回 queue.ErrClosed。
func (q *Queue[T]) Dequeue() (T, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for q.elements.Len() == 0 && !q.closed {
		q.cond.Wait()
	}

	return q.tryDequeue()
}

func (q *Queue[T]) tryDequeue() (T, error) {
	if q.elements.Len() == 0 {
		var zero T
		if q.closed {
			return zero, queue.ErrClosed
		}
		return zero, queue.ErrEmpty
	}

	return heap.Pop(q.elements).(T), nil
}

// Close 关闭队列。
// 关闭后不再接受新元素，但允许取出已有元素。
func (q *Queue[T]) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return
	}

	q.closed = true
	q.cond.Broadcast()
}

// Closed 判断队列是否已关闭。
func (q *Queue[T]) Closed() bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.closed
}
