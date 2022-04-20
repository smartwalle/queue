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
type Queue interface {
	// Now 获取队列使用的当前时间
	Now() int64

	// Len 获取队列元素数量
	Len() int

	// Enqueue 添加元素到队列
	// 参数 expiration 的值不能小于 0
	Enqueue(value interface{}, expiration int64)

	// Dequeue 获取队列中已过期的元素及其过期时间，并且将该元素从队列中删除
	// 如果队列中没有过期的元素，则本方法会一直阻塞，直到有过期的元素
	// 如果队列被关闭，则返回 nil 和 -1
	Dequeue() (interface{}, int64)

	// Close 关闭延迟队列
	Close()
}

type delayQueue struct {
	*option

	mu sync.Mutex
	pq priority.Queue

	sleeping int32
	wakeup   chan struct{}
	close    chan struct{}
}

func New(opts ...Option) Queue {
	var q = &delayQueue{}
	q.option = &option{
		unit: time.Second,
		timer: func() int64 {
			return time.Now().Unix()
		},
	}
	for _, opt := range opts {
		opt(q.option)
	}

	q.pq = priority.New()
	q.wakeup = make(chan struct{})
	q.close = make(chan struct{})

	return q
}

func (dq *delayQueue) Now() int64 {
	return dq.option.timer()
}

func (dq *delayQueue) Len() int {
	return dq.pq.Len()
}

func (dq *delayQueue) Enqueue(value interface{}, expiration int64) {
	select {
	case <-dq.close:
	default:

		dq.mu.Lock()
		var nItem = dq.pq.Enqueue(value, expiration)
		dq.mu.Unlock()

		if nItem.Index() == 0 {
			if atomic.CompareAndSwapInt32(&dq.sleeping, 1, 0) {
				dq.wakeup <- struct{}{}
			}
		}
	}
}

func (dq *delayQueue) Dequeue() (interface{}, int64) {
	var value interface{}
	var expiration int64
	var delay int64

ReadLoop:
	for {
		var nTime = dq.option.timer()

		dq.mu.Lock()
		value, expiration, delay = dq.pq.Peek(nTime)
		if value == nil {
			atomic.StoreInt32(&dq.sleeping, 1)
		}
		dq.mu.Unlock()

		if value == nil {
			if delay == 0 {
				select {
				case <-dq.wakeup:
					continue
				case <-dq.close:
					break ReadLoop
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
				case <-dq.close:
					timer.Stop()
					break ReadLoop
				}
			}
		}

		break ReadLoop
	}

	select {
	case <-dq.close:
		value = nil
		expiration = -1
	default:
	}

	atomic.StoreInt32(&dq.sleeping, 0)
	return value, expiration
}

func (dq *delayQueue) Close() {
	close(dq.close)
}
