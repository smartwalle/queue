package block

import (
	"sync"
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
	Enqueue(value interface{}) bool

	// Dequeue 获取队列中的所有元素
	// 如果队列中没有元素，则本方法会一直阻塞，直到有元素
	// 如果队列被关闭，会添加一个 nil 到元素列表的结尾，调用者可以根据是否获取到 nil 元素来判断队列是否关闭
	Dequeue(*[]interface{})

	// Close 关闭队列
	Close()

	// Closed 获取队列是否关闭
	Closed() bool
}

type blockQueue struct {
	*option
	elements []interface{}
	cond     *sync.Cond
	close    chan struct{}
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
	q.close = make(chan struct{})
	return q
}

func (bq *blockQueue) Enqueue(value interface{}) bool {
	select {
	case <-bq.close:
		return false
	default:
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

		//bq.elements = append(bq.elements, value)
		bq.cond.L.Unlock()
		bq.cond.Signal()
		return true
	}
}

func (bq *blockQueue) Dequeue(elements *[]interface{}) {
	bq.cond.L.Lock()

	for len(bq.elements) == 0 {
		select {
		case <-bq.close:
			bq.cond.L.Unlock()
			*elements = append(*elements, nil)
			return
		default:
			bq.cond.Wait()
		}
	}

	for _, ele := range bq.elements {
		*elements = append(*elements, ele)
		if ele == nil {
			break
		}
	}

	bq.elements = bq.elements[0:0]
	bq.cond.L.Unlock()
	bq.cond.Signal()
}

func (bq *blockQueue) Close() {
	select {
	case <-bq.close:
	default:
		close(bq.close)
		bq.cond.Signal()
	}
}

func (bq *blockQueue) Closed() bool {
	select {
	case <-bq.close:
		return true
	default:
		return false
	}
}
