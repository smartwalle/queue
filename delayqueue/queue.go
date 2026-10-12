package delayqueue

import (
	"sync"
	"time"

	"github.com/smartwalle/queue"
)

type delayed[T any] struct {
	elem     T
	deadline time.Time
}

// delayHeap 是基于 deadline 排序的最小堆，队首元素具有最早的截止时间。
type delayHeap[T any] struct {
	elements []delayed[T]
}

// Len 返回堆中元素的数量。
func (h *delayHeap[T]) Len() int {
	return len(h.elements)
}

// less 判断第 i 个元素的截止时间是否早于第 j 个元素的截止时间。
func (h *delayHeap[T]) less(i, j int) bool {
	return h.elements[i].deadline.Before(h.elements[j].deadline)
}

// swap 交换堆中第 i 个和第 j 个元素。
func (h *delayHeap[T]) swap(i, j int) {
	h.elements[i], h.elements[j] = h.elements[j], h.elements[i]
}

// push 将元素添加到堆中，并向上调整以维持堆的性质。
func (h *delayHeap[T]) push(value delayed[T]) {
	h.elements = append(h.elements, value)
	h.up(len(h.elements) - 1)
}

// pop 移除并返回截止时间最早的元素，并向下调整以维持堆的性质。
func (h *delayHeap[T]) pop() delayed[T] {
	n := len(h.elements) - 1
	h.swap(0, n)
	h.down(0, n)

	value := h.elements[n]

	var zero delayed[T]
	h.elements[n] = zero
	h.elements = h.elements[:n]

	return value
}

// up 从指定位置开始向上调整堆，直到满足最小堆的性质。
func (h *delayHeap[T]) up(i int) {
	for {
		parent := (i - 1) / 2
		if parent == i || !h.less(i, parent) {
			return
		}

		h.swap(parent, i)
		i = parent
	}
}

// down 从指定位置开始向下调整堆，使索引范围 [0, n) 满足最小堆的性质。
func (h *delayHeap[T]) down(i, n int) {
	for {
		left := 2*i + 1
		if left >= n {
			return
		}

		j := left
		right := left + 1

		if right < n && h.less(right, left) {
			j = right
		}

		if !h.less(j, i) {
			return
		}

		h.swap(i, j)
		i = j
	}
}

// Queue 延时队列。
type Queue[T any] struct {
	mu       sync.Mutex
	notEmpty *sync.Cond
	notFull  *sync.Cond

	capacity int
	elements delayHeap[T]
	deadline func(T) time.Time
	closed   bool
}

// New 创建延时队列，并使用 deadline 计算每个元素的截止时间。
// capacity > 0 时为有界队列；capacity <= 0 时为无界队列。
func New[T any](capacity int, deadline func(T) time.Time) *Queue[T] {
	var q = &Queue[T]{
		elements: delayHeap[T]{
			elements: make([]delayed[T], 0, max(32, capacity)),
		},
		deadline: deadline,
		capacity: capacity,
	}

	q.notEmpty = sync.NewCond(&q.mu)
	q.notFull = sync.NewCond(&q.mu)

	return q
}

// Len 返回队列中尚未出队的元素数量，包括尚未到期的元素。
func (q *Queue[T]) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.elements.Len()
}

// Enqueue 将元素加入延时队列。
func (q *Queue[T]) Enqueue(value T) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for q.capacity > 0 && q.elements.Len() >= q.capacity && !q.closed {
		q.notFull.Wait()
	}

	if q.closed {
		return queue.ErrClosed
	}

	q.elements.push(delayed[T]{
		elem:     value,
		deadline: q.deadline(value),
	})
	q.notEmpty.Broadcast()

	return nil
}

// TryDequeue 尝试移除并返回队首元素。
// 队列为空或队首元素尚未到期时返回 queue.ErrEmpty；
// 队列已关闭且为空时返回 queue.ErrClosed。
func (q *Queue[T]) TryDequeue() (T, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.elements.Len() == 0 {
		var zero T
		if q.closed {
			return zero, queue.ErrClosed
		}
		return zero, queue.ErrEmpty
	}

	if time.Until(q.elements.elements[0].deadline) > 0 {
		var zero T
		return zero, queue.ErrEmpty
	}

	value := q.elements.pop().elem
	if q.capacity > 0 {
		q.notFull.Signal()
	}

	return value, nil
}

// Dequeue 阻塞等待队首元素到期，并移除和返回该元素。
// 队列为空时等待新元素入队；队首元素尚未到期时等待其到期。
// 队列关闭后仍会处理已有元素，直到队列排空后返回 queue.ErrClosed。
func (q *Queue[T]) Dequeue() (T, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for {
		if q.elements.Len() == 0 {
			if q.closed {
				var zero T
				return zero, queue.ErrClosed
			}

			q.notEmpty.Wait()
			continue
		}

		delay := time.Until(q.elements.elements[0].deadline)
		if delay <= 0 {
			value := q.elements.pop().elem
			if q.capacity > 0 {
				q.notFull.Signal()
			}
			return value, nil
		}

		timer := time.AfterFunc(delay, func() {
			q.mu.Lock()
			q.notEmpty.Broadcast()
			q.mu.Unlock()
		})

		q.notEmpty.Wait()
		timer.Stop()
	}
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
	q.notEmpty.Broadcast()
	q.notFull.Broadcast()
}

// Closed 返回队列是否已关闭。
func (q *Queue[T]) Closed() bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.closed
}
