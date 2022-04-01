package delay

import (
	"container/heap"
	"github.com/smartwalle/queue/priority"
	"sync"
	"sync/atomic"
	"time"
)

type Option func(opt *option)

func WithCapacity(capacity int) Option {
	return func(opt *option) {
		if capacity <= 0 {
			capacity = 32
		}
		opt.capacity = capacity
	}
}

func WithTimeUnit(unit time.Duration) Option {
	return func(opt *option) {
		if unit <= 0 {
			unit = time.Millisecond
		}
		opt.unit = unit
	}
}

func WithTimeProvider(f func() int64) Option {
	return func(opt *option) {
		if f == nil {
			f = func() int64 {
				return time.Now().UnixMilli()
			}
		}
		opt.timer = f
	}
}

type option struct {
	capacity int
	unit     time.Duration
	timer    func() int64
}

type Queue struct {
	*option

	mu sync.Mutex
	pq priority.Queue

	sleeping int32
	wakeup   chan struct{}
	close    chan struct{}
}

func New(opts ...Option) *Queue {
	var q = &Queue{}
	q.option = &option{
		capacity: 32,
		unit:     time.Millisecond,
	}
	for _, opt := range opts {
		opt(q.option)
	}

	q.pq = priority.New(q.option.capacity)
	q.wakeup = make(chan struct{})
	q.close = make(chan struct{})

	return q
}

func (dq *Queue) Push(x interface{}, expiration int64) {
	select {
	case <-dq.close:
	default:
		var item = &priority.Item{Value: x, Priority: expiration}

		dq.mu.Lock()
		heap.Push(&dq.pq, item)
		index := item.Index
		dq.mu.Unlock()

		if index == 0 {
			if atomic.CompareAndSwapInt32(&dq.sleeping, 1, 0) {
				dq.wakeup <- struct{}{}
			}
		}
	}
}

func (dq *Queue) Pop() interface{} {
	var result interface{}

ReadLoop:
	for {
		var nTime = dq.option.timer()

		dq.mu.Lock()
		item, delay := dq.pq.PeekAndShift(nTime)
		if item == nil {
			atomic.StoreInt32(&dq.sleeping, 1)
		}
		dq.mu.Unlock()

		if item == nil {
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

		select {
		case <-dq.close:
		default:
			result = item.Value
		}
		break ReadLoop
	}

	atomic.StoreInt32(&dq.sleeping, 0)
	return result
}

func (dq *Queue) Close() {
	close(dq.close)
}
