package block

import (
	"sync"
	"sync/atomic"
)

type Option func(opt *option)

func WithMaxSize(max int) Option {
	return func(opt *option) {
		opt.max = max
	}
}

type option struct {
	max int
}

// Queue 阻塞队列
type Queue interface {
	// Enqueue 添加元素到队列
	// 如果队列已关闭，则返回 false，否则返回 true
	Enqueue(value interface{}) bool

	// Dequeue 获取队列中的所有元素
	// 如果队列中没有元素，则本方法会一直阻塞，直到有元素
	// 如果队列已关闭，则返回 false，否则返回 true
	Dequeue(*[]interface{}) bool

	// Close 关闭队列
	Close()

	// Closed 获取队列是否关闭
	Closed() bool
}

type blockQueue struct {
	*option
	elements []interface{}
	cond     *sync.Cond
	closed   int32
}

func New(opts ...Option) Queue {
	var q = &blockQueue{}
	q.option = &option{}
	for _, opt := range opts {
		if opt != nil {
			opt(q.option)
		}
	}
	q.elements = make([]interface{}, 0, 32)
	q.cond = sync.NewCond(&sync.Mutex{})
	return q
}

func (bq *blockQueue) Enqueue(value interface{}) bool {
	if atomic.LoadInt32(&bq.closed) == 1 {
		return false
	}

	bq.cond.L.Lock()
	if bq.option.max > 0 && len(bq.elements)+1 > bq.option.max {
		bq.cond.Wait()
	}

	n := len(bq.elements)
	c := cap(bq.elements)
	if n+1 > c {
		npq := make([]interface{}, n, c*2)
		copy(npq, bq.elements)
		bq.elements = npq
	}
	bq.elements = bq.elements[0 : n+1]
	bq.elements[n] = value

	bq.cond.L.Unlock()
	bq.cond.Signal()
	return true
}

func (bq *blockQueue) Dequeue(elements *[]interface{}) bool {
	if atomic.LoadInt32(&bq.closed) == 1 {
		return false
	}

	bq.cond.L.Lock()

	for len(bq.elements) == 0 {
		if atomic.LoadInt32(&bq.closed) == 1 {
			bq.cond.L.Unlock()
			return false
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

func (bq *blockQueue) Close() {
	if atomic.CompareAndSwapInt32(&bq.closed, 0, 1) {
		bq.cond.Signal()
	}
}

func (bq *blockQueue) Closed() bool {
	return atomic.LoadInt32(&bq.closed) == 1
}
