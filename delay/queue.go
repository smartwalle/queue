package delay

import (
	"github.com/smartwalle/queue/priority"
	"sync"
	"sync/atomic"
	"time"
)

type Option func(opts *options)

// WithTimeUnit 用于设定队列时间单位
func WithTimeUnit(unit time.Duration) Option {
	return func(opts *options) {
		if unit <= 0 {
			unit = time.Second
		}
		opts.unit = unit
	}
}

// WithTimeProvider 用于设定队列的时间源
func WithTimeProvider(f func() int64) Option {
	return func(opts *options) {
		if f == nil {
			f = func() int64 {
				return time.Now().Unix()
			}
		}
		opts.clock = f
	}
}

// WithDrainAll 调用队列的 Close 方法后，队列会等到所有的消息都出队后才关闭，但是不能再往队列添加消息或执行其它更新操作
func WithDrainAll() Option {
	return func(opts *options) {
		opts.drainAll = true
	}
}

type options struct {
	clock    func() int64
	unit     time.Duration
	drainAll bool
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
	pq       priority.Queue[T]
	empty    T
	options  *options
	wakeup   chan struct{}
	mu       sync.Mutex
	w        sync.WaitGroup
	timer    *time.Timer
	sleeping int32
	closed   bool
}

func New[T any](opts ...Option) Queue[T] {
	var q = &delayQueue[T]{}
	q.options = &options{
		unit: time.Second,
		clock: func() int64 {
			return time.Now().Unix()
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(q.options)
		}
	}
	q.pq = priority.New[T]()
	q.wakeup = make(chan struct{})
	return q
}

func (dq *delayQueue[T]) Len() int {
	dq.mu.Lock()
	defer dq.mu.Unlock()
	return dq.pq.Len()
}

func (dq *delayQueue[T]) Enqueue(value T, expiration int64) priority.Element {
	dq.mu.Lock()
	if dq.closed {
		dq.mu.Unlock()
		return nil
	}

	var ele = dq.pq.Enqueue(value, expiration)
	dq.mu.Unlock()

	if ele != nil && ele.First() {
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
	var done bool

ReadLoop:
	for {
		dq.mu.Lock()

		if dq.closed && (!dq.options.drainAll || dq.pq.Len() == 0) {
			done = true
			dq.mu.Unlock()
			break ReadLoop
		}

		var nTime = dq.options.clock()
		value, expiration, delay, found = dq.pq.Peek(nTime)
		if !found {
			atomic.StoreInt32(&dq.sleeping, 1)
		}

		if found && dq.closed {
			dq.w.Done()
		}

		dq.mu.Unlock()

		if !found {
			if delay == 0 {
				select {
				case <-dq.wakeup:
					continue
				}
			} else if delay > 0 {
				if dq.timer == nil {
					dq.timer = time.NewTimer(time.Duration(delay) * dq.options.unit)
				} else {
					stopTimer(dq.timer)
					dq.timer.Reset(time.Duration(delay) * dq.options.unit)
				}

				select {
				case <-dq.wakeup:
					stopTimer(dq.timer)
					continue
				case <-dq.timer.C:
					if atomic.SwapInt32(&dq.sleeping, 0) == 0 {
						<-dq.wakeup
					}
					continue
				}
			}
		}

		break ReadLoop
	}

	if done {
		value = dq.empty
		expiration = -1
	}

	atomic.StoreInt32(&dq.sleeping, 0)
	return value, expiration
}

func (dq *delayQueue[T]) Update(ele priority.Element, expiration int64) {
	dq.mu.Lock()
	if dq.closed {
		dq.mu.Unlock()
		return
	}

	dq.pq.Update(ele, expiration)
	dq.mu.Unlock()

	if ele.First() {
		if atomic.CompareAndSwapInt32(&dq.sleeping, 1, 0) {
			dq.wakeup <- struct{}{}
		}
	}
}

func (dq *delayQueue[T]) Remove(ele priority.Element) {
	var first = false
	if ele != nil {
		first = ele.First()
	}

	dq.mu.Lock()
	if dq.closed {
		dq.mu.Unlock()
		return
	}

	dq.pq.Remove(ele)
	dq.mu.Unlock()

	if first {
		if atomic.CompareAndSwapInt32(&dq.sleeping, 1, 0) {
			dq.wakeup <- struct{}{}
		}
	}
}

func (dq *delayQueue[T]) Close() {
	dq.mu.Lock()
	if dq.closed {
		dq.mu.Unlock()
		return
	}

	dq.closed = true

	if atomic.LoadInt32(&dq.sleeping) == 1 {
		dq.wakeup <- struct{}{}
	}

	if dq.options.drainAll {
		var c = dq.pq.Len()
		if c > 0 {
			dq.w.Add(c)
		}
		dq.mu.Unlock()

		dq.w.Wait()
	} else {
		dq.mu.Unlock()
	}
}

func (dq *delayQueue[T]) Closed() bool {
	dq.mu.Lock()
	defer dq.mu.Unlock()
	return dq.closed
}

func stopTimer(timer *time.Timer) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}
