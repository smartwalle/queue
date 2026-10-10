package queue

import "errors"

var ErrClosed = errors.New("queue already closed")

var ErrEmpty = errors.New("queue is empty")

// Queue 队列
type Queue[T any] interface {
	// Len 获取队列当前的元素数量。
	Len() int

	// Enqueue 添加元素到队列
	Enqueue(T) error

	// TryDequeue 尝试获取队首的元素，不会阻塞。
	// 如果队列为空，则返回错误。
	TryDequeue() (T, error)

	// Dequeue 获取队首的元素。
	// 如果队列为空，则阻塞等待；如果队列已关闭且没有剩余元素，则返回错误。
	Dequeue() (T, error)

	// Close 关闭队列
	Close()

	// Closed 获取队列是否关闭
	Closed() bool
}
