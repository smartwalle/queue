package queue

import "errors"

var ErrClosed = errors.New("queue already closed")

// Queue 队列
type Queue[T any] interface {
	// Len 获取队列当前的元素数量。
	Len() int

	// Enqueue 添加元素到队列
	Enqueue(T) error

	// Dequeue 获取队首的一个元素
	Dequeue() (T, error)

	// Close 关闭队列
	Close()

	// Closed 获取队列是否关闭
	Closed() bool
}
