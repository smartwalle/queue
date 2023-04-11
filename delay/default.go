package delay

import (
	"sync/atomic"
	"time"
)

type defaultMode[T any] struct {
}

func newDefaultMode[T any]() *defaultMode[T] {
	return &defaultMode[T]{}
}

func (m *defaultMode[T]) dequeue(dq *delayQueue[T]) (T, int64) {
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

		var nTime = dq.options.clock()

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

func (m *defaultMode[T]) close(dq *delayQueue[T]) {

}
