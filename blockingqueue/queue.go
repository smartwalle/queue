package blockingqueue

import (
	"sync"

	"github.com/smartwalle/queue"
)

// Queue 阻塞队列。
type Queue[T any] struct {
	mu       sync.Mutex
	notEmpty *sync.Cond
	notFull  *sync.Cond

	max      int
	elements []T
	head     int
	closed   bool
}

// New 创建阻塞队列。
// max > 0 时为有界队列；max <= 0 时为无界队列。
func New[T any](max int) *Queue[T] {
	var q = &Queue[T]{
		max:      max,
		elements: make([]T, 0, 32),
	}

	q.notEmpty = sync.NewCond(&q.mu)
	q.notFull = sync.NewCond(&q.mu)

	return q
}

// Len 返回队列中尚未出队的元素数量。
func (q *Queue[T]) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return len(q.elements) - q.head
}

// Enqueue 将元素加入队列。
func (q *Queue[T]) Enqueue(value T) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for !q.closed &&
		q.max > 0 &&
		len(q.elements)-q.head >= q.max {
		q.notFull.Wait()
	}

	if q.closed {
		return queue.ErrClosed
	}

	q.elements = append(q.elements, value)
	q.notEmpty.Signal()

	return nil
}

// TryDequeue 尝试从队列中取出一个元素，不会阻塞。
// 如果队列为空且已关闭，返回 queue.ErrClosed。
// 如果队列为空且未关闭，返回 queue.ErrEmpty。
func (q *Queue[T]) TryDequeue() (T, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.tryDequeue()
}

// Dequeue 从队列中取出一个元素。
// 队列为空时阻塞等待。
// 队列关闭后，仍然允许取出剩余元素。
// 队列关闭且为空时，返回 queue.ErrClosed。
func (q *Queue[T]) Dequeue() (T, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for q.head == len(q.elements) && !q.closed {
		q.notEmpty.Wait()
	}

	return q.tryDequeue()
}

func (q *Queue[T]) tryDequeue() (T, error) {
	var zero T

	if q.head == len(q.elements) {
		if q.closed {
			return zero, queue.ErrClosed
		}
		return zero, queue.ErrEmpty
	}

	value := q.elements[q.head]
	q.elements[q.head] = zero
	q.head++

	q.notFull.Signal()

	if q.head == len(q.elements) {
		q.elements = q.elements[:0]
		q.head = 0
	} else if q.head >= 1024 && q.head*2 >= len(q.elements) {
		n := copy(q.elements, q.elements[q.head:])
		clear(q.elements[n:])
		q.elements = q.elements[:n]
		q.head = 0
	}

	return value, nil
}

// Close 关闭队列，不等待剩余元素出队。
// 关闭后不再接受新元素，但允许取出已有元素。
func (q *Queue[T]) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return
	}

	q.closed = true
	q.notEmpty.Broadcast()
	q.notFull.Broadcast()
}

// Closed 返回队列是否已关闭。
func (q *Queue[T]) Closed() bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.closed
}
