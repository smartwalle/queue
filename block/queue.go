package block

import (
	"sync"
)

type Queue interface {
	Enqueue(value interface{})

	Dequeue(*[]interface{})

	Close()
}

type blockQueue struct {
	elements []interface{}
	cond     *sync.Cond
	close    chan struct{}
}

func New() Queue {
	var q = &blockQueue{}
	q.cond = sync.NewCond(&sync.Mutex{})
	q.close = make(chan struct{})
	return q
}

func (bq *blockQueue) Enqueue(value interface{}) {
	select {
	case <-bq.close:
	default:
		bq.cond.L.Lock()
		bq.elements = append(bq.elements, value)
		bq.cond.L.Unlock()

		bq.cond.Signal()
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
}

func (bq *blockQueue) Close() {
	select {
	case <-bq.close:
	default:
		bq.cond.L.Lock()
		close(bq.close)
		bq.cond.L.Unlock()

		bq.cond.Signal()
	}
}
