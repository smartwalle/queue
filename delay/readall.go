package delay

import (
	"sync"
	"sync/atomic"
	"time"
)

func newReadAllMode[T any]() *readAllMode[T] {
	var m = &readAllMode[T]{}
	return m
}

type readAllMode[T any] struct {
	w *sync.WaitGroup
}

func (m *readAllMode[T]) dequeue(dq *delayQueue[T]) (T, int64) {
	var value T
	var expiration int64
	var delay int64
	var found bool
	var isClose bool

ReadLoop:
	for {
		var nTime = dq.option.timer()

		dq.mu.Lock()
		value, expiration, delay, found = dq.pq.Peek(nTime)
		if found == false {
			atomic.StoreInt32(&dq.sleeping, 1)
		}

		if found && m.w != nil {
			m.w.Done()
		}

		dq.mu.Unlock()

		if found == false {
			if delay == 0 {
				if expiration == -1 {
					isClose = true
					break ReadLoop
				}

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

func (m *readAllMode[T]) close(dq *delayQueue[T]) {
	dq.mu.Lock()
	var c = dq.pq.Len()
	if c > 0 {
		m.w = &sync.WaitGroup{}
		m.w.Add(c)
	}
	dq.mu.Unlock()

	if m.w != nil {
		m.w.Wait()
	}
}
