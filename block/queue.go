package block

import (
	"sync"
	"sync/atomic"
)

type Option func(opts *options)

func WithMaxSize(max int) Option {
	return func(opts *options) {
		opts.max = max
	}
}

type options struct {
	max int
}

// Queue 阻塞队列
type Queue[T any] interface {
	// Enqueue 添加元素到队列
	// 如果队列已关闭，则返回 false，否则返回 true
	Enqueue(value T) bool

	// Dequeue 获取队列中的所有元素
	// 如果队列中没有元素，则本方法会一直阻塞，直到有元素
	// 如果队列已关闭，则返回 false，否则返回 true
	Dequeue(*[]T) bool

	// Close 关闭队列
	Close()

	// Closed 获取队列是否关闭
	Closed() bool
}

type blockQueue[T any] struct {
	options  *options
	cond     *sync.Cond
	elements []T
	closed   int32
}

func New[T any](opts ...Option) Queue[T] {
	var q = &blockQueue[T]{}
	q.options = &options{}
	for _, opt := range opts {
		if opt != nil {
			opt(q.options)
		}
	}
	q.elements = make([]T, 0, 32)
	q.cond = sync.NewCond(&sync.Mutex{})
	return q
}

func (bq *blockQueue[T]) Enqueue(value T) bool {
	if atomic.LoadInt32(&bq.closed) == 1 {
		return false
	}

	bq.cond.L.Lock()
	if bq.options.max > 0 && len(bq.elements)+1 > bq.options.max {
		bq.cond.Wait()
	}

	n := len(bq.elements)
	c := cap(bq.elements)
	if n+1 > c {
		npq := make([]T, n, c*2)
		copy(npq, bq.elements)
		bq.elements = npq
	}
	bq.elements = bq.elements[0 : n+1]
	bq.elements[n] = value

	bq.cond.L.Unlock()
	bq.cond.Signal()
	return true
}

func (bq *blockQueue[T]) Dequeue(elements *[]T) bool {
	//if atomic.LoadInt32(&bq.closed) == 1 {
	//	return false
	//}

	bq.cond.L.Lock()

	for len(bq.elements) == 0 {
		if atomic.LoadInt32(&bq.closed) == 1 {
			break
		}
		bq.cond.Wait()
	}

	for _, ele := range bq.elements {
		*elements = append(*elements, ele)
	}

	bq.elements = bq.elements[0:0]
	bq.cond.L.Unlock()
	bq.cond.Signal()
	return atomic.LoadInt32(&bq.closed) != 1
}

func (bq *blockQueue[T]) Close() {
	if atomic.CompareAndSwapInt32(&bq.closed, 0, 1) {
		bq.cond.Broadcast()
	}
}

func (bq *blockQueue[T]) Closed() bool {
	return atomic.LoadInt32(&bq.closed) == 1
}
