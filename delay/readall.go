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
	closed bool
	w      sync.WaitGroup
}

func (m *readAllMode[T]) dequeue(dq *delayQueue[T]) (T, int64) {
	var value T
	var expiration int64
	var delay int64
	var found bool
	var isClose bool

ReadLoop:
	for {
		var nTime = dq.options.clock()

		dq.mu.Lock()

		if dq.pq.Len() == 0 && atomic.LoadInt32(&dq.closed) == 1 {
			isClose = true
			dq.mu.Unlock()
			break ReadLoop
		}

		value, expiration, delay, found = dq.pq.Peek(nTime)
		if found == false {
			atomic.StoreInt32(&dq.sleeping, 1)
		}

		if found && m.closed {
			m.w.Done()
		}

		dq.mu.Unlock()

		if found == false {
			if delay == 0 {
				select {
				case <-dq.wakeup:
					continue
				}
			} else if delay > 0 {
				var timer = time.NewTimer(time.Duration(delay) * dq.options.unit)
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
	if m.closed {
		dq.mu.Unlock()
		return
	}

	m.closed = true
	var c = dq.pq.Len()
	if c > 0 {
		m.w.Add(c)
	}
	dq.mu.Unlock()

	m.w.Wait()
}
