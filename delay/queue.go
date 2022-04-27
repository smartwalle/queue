package delay

import (
	"github.com/smartwalle/queue/priority"
	"sync"
	"sync/atomic"
	"time"
)

type Option func(opt *option)

// WithTimeUnit 用于设定队列时间单位
func WithTimeUnit(unit time.Duration) Option {
	return func(opt *option) {
		if unit <= 0 {
			unit = time.Second
		}
		opt.unit = unit
	}
}

// WithTimeProvider 用于设定队列的时间源
func WithTimeProvider(f func() int64) Option {
	return func(opt *option) {
		if f == nil {
			f = func() int64 {
				return time.Now().Unix()
			}
		}
		opt.timer = f
	}
}

type option struct {
	unit  time.Duration
	timer func() int64
}

// Queue 延迟队列
type Queue[T any] interface {
	// Len 获取队列元素数量
	Len() int

	// Enqueue 添加元素到队列
	// 参数 expiration 的值不能小于 0
	// 如果队列已关闭，则返回 nil
	Enqueue(value T, expiration int64) priority.Element

	// Dequeue 获取队列中已过期的元素及其过期时间，并且将该元素从队列中删除
	// 如果队列中没有过期的元素，则本方法会一直阻塞，直到有过期的元素
	// 如果队列被关闭，则返回空值和 -1
	Dequeue() (T, int64)

	// Update 更新元素的过期时间
	Update(ele priority.Element, expiration int64)

	// Remove 从队列中删除元素
	Remove(ele priority.Element)

	// Close 关闭队列
	Close()

	// Closed 获取队列是否关闭
	Closed() bool
}

type delayQueue[T any] struct {
	*option

	mu sync.Mutex
	pq priority.Queue[T]

	sleeping int32
	wakeup   chan struct{}
	closed   int32
	empty    T
}

func New[T any](opts ...Option) Queue[T] {
	var q = &delayQueue[T]{}
	q.option = &option{
		unit: time.Second,
		timer: func() int64 {
			return time.Now().Unix()
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(q.option)
		}
	}

	q.pq = priority.New[T]()
	q.wakeup = make(chan struct{})
	return q
}

func (dq *delayQueue[T]) Len() int {
	return dq.pq.Len()
}

func (dq *delayQueue[T]) Enqueue(value T, expiration int64) priority.Element {
	if atomic.LoadInt32(&dq.closed) == 1 {
		return nil
	}

	dq.mu.Lock()
	var ele = dq.pq.Enqueue(value, expiration)
	dq.mu.Unlock()

	if ele != nil && ele.IsFirst() {
		if atomic.CompareAndSwapInt32(&dq.sleeping, 1, 0) {
			dq.wakeup <- struct{}{}
		}
	}
	return ele
}

func (dq *delayQueue[T]) Dequeue() (T, int64) {
	var value T
	var expiration int64
	var delay int64
	var found bool
	var isClose bool

ReadLoop:
	for {
		if atomic.LoadInt32(&dq.closed) == 1 {
			isClose = true
			break ReadLoop
		}

		var nTime = dq.option.timer()

		dq.mu.Lock()
		value, expiration, delay, found = dq.pq.Peek(nTime)
		if found == false {
			atomic.StoreInt32(&dq.sleeping, 1)
		}
		dq.mu.Unlock()

		if found == false {
			if delay == 0 {
				select {
				case <-dq.wakeup:
					continue
				}
			} else if delay > 0 {
				var timer = time.NewTimer(time.Duration(delay) * dq.option.unit)
				select {
				case <-dq.wakeup:
					timer.Stop()
					continue
				case <-timer.C:
					if atomic.SwapInt32(&dq.sleeping, 0) == 0 {
						<-dq.wakeup
					}
					continue
				}
			}
		}

		break ReadLoop
	}

	if isClose {
		value = dq.empty
		expiration = -1
	}

	atomic.StoreInt32(&dq.sleeping, 0)
	return value, expiration
}

func (dq *delayQueue[T]) Update(ele priority.Element, expiration int64) {
	if atomic.LoadInt32(&dq.closed) == 1 {
		return
	}

	dq.mu.Lock()
	dq.pq.Update(ele, expiration)
	dq.mu.Unlock()

	if ele.IsFirst() {
		if atomic.CompareAndSwapInt32(&dq.sleeping, 1, 0) {
			dq.wakeup <- struct{}{}
		}
	}
}
func (dq *delayQueue[T]) Remove(ele priority.Element) {
	if atomic.LoadInt32(&dq.closed) == 1 {
		return
	}

	var isFirst = false
	if ele != nil {
		isFirst = ele.IsFirst()
	}
	dq.mu.Lock()
	dq.pq.Remove(ele)
	dq.mu.Unlock()

	if isFirst {
		if atomic.CompareAndSwapInt32(&dq.sleeping, 1, 0) {
			dq.wakeup <- struct{}{}
		}
	}
}

func (dq *delayQueue[T]) Close() {
	if atomic.CompareAndSwapInt32(&dq.closed, 0, 1) {
		dq.wakeup <- struct{}{}
	}
}

func (dq *delayQueue[T]) Closed() bool {
	return atomic.LoadInt32(&dq.closed) == 1
}
