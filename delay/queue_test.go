package delay_test

import (
	"github.com/smartwalle/queue/delay"
	"github.com/smartwalle/queue/priority"
	"math/rand"
	"testing"
	"time"
)

func BenchmarkDelayQueue_Enqueue(b *testing.B) {
	var q = delay.New[int]()

	for i := 0; i < b.N; i++ {
		q.Enqueue(i, int64(i))
	}
}

func BenchmarkDelayQueue_EnqueueDequeue(b *testing.B) {
	var q = delay.New[int](delay.WithDrainAll())
	go func() {
		var next int64
		for {
			_, next = q.Dequeue()
			if next < 0 {
				break
			}
		}
	}()

	for i := 0; i < b.N; i++ {
		q.Enqueue(i, 1)
	}
	q.Close()
}

func BenchmarkDelayQueue_Dequeue(b *testing.B) {
	var q = delay.New[int]()
	var now = time.Now().Unix()
	for i := 0; i < b.N; i++ {
		q.Enqueue(i, now)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, expiration := q.Dequeue(); expiration == -1 {
			b.Fatal("Dequeue 异常")
		}
	}
}

func BenchmarkDelayQueue_DrainAll_Dequeue(b *testing.B) {
	var q = delay.New[int](delay.WithDrainAll())
	var now = time.Now().Unix()
	for i := 0; i < b.N; i++ {
		q.Enqueue(i, now)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, expiration := q.Dequeue(); expiration == -1 {
			b.Fatal("Dequeue 异常")
		}
	}
}

func BenchmarkDelayQueue_Remove(b *testing.B) {
	var q = delay.New[int]()

	var r = rand.NewSource(time.Now().Unix())
	var elements = make([]priority.Element, b.N)
	for i := 0; i < b.N; i++ {
		var p = r.Int63()
		var ele = q.Enqueue(i, p)
		elements[i] = ele
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Remove(elements[i])
	}
}

func TestDelayQueue_Close1(t *testing.T) {
	var q = delay.New[int]()

	go func() {
		for {
			var value, expiration = q.Dequeue()
			if expiration == -1 {
				return
			}
			t.Log("Dequeue", value)
		}
	}()

	var now = time.Now().Unix()
	q.Enqueue(1, now+1)
	q.Enqueue(2, now+2)
	q.Enqueue(3, now+3)

	// 调用 Close 方法后立即关闭队列，理想情况下不会有任何消息被消费
	q.Close()

	// Close 方法不会阻塞，所以基本没有时间差
	if time.Now().Unix() > now+1 {
		t.Fatal("Close 方法消耗时间异常")
	}
}

func TestDelayQueue_DrainAll_Close1(t *testing.T) {
	var q = delay.New[int](delay.WithDrainAll())

	var done = make(chan struct{})
	go func() {
		defer func() {
			close(done)
		}()
		for {
			var value, expiration = q.Dequeue()
			if expiration == -1 {
				return
			}
			t.Log("Dequeue", value)
		}
	}()

	var now = time.Now().Unix()
	q.Enqueue(1, now+1)
	q.Enqueue(2, now+2)
	q.Enqueue(3, now+3)

	// DrainAll：调用 Close 方法后会等待所有已入队的消息出队，所以所有的消息都会被消费
	q.Close()

	select {
	case <-done:
	}

	// Close 方法会阻塞，所以时间差基本是最后一条消息的延迟时间
	if time.Now().Unix() < now+3 {
		t.Fatal("Close 方法消耗时间异常")
	}
}

func TestDelayQueue_Close2(t *testing.T) {
	var q = delay.New[int]()

	var now = time.Now().Unix()
	q.Enqueue(1, now+1)
	q.Enqueue(2, now+2)
	q.Enqueue(3, now+3)

	q.Close()

	if _, exp := q.Dequeue(); exp != -1 {
		t.Fatal("队列已关闭，Dequeue 获取到的过期时间应该是 -1")
	}

	if ele := q.Enqueue(4, now+4); ele != nil {
		t.Fatal("队列已关闭，Enqueue 的返回值应该是 nil")
	}
}

func TestDelayQueue_DrainAll_Close2(t *testing.T) {
	var q = delay.New[int](delay.WithDrainAll())

	var now = time.Now().Unix()
	q.Enqueue(1, now+1)
	q.Enqueue(2, now+2)
	q.Enqueue(3, now+3)

	q.Dequeue()
	q.Dequeue()
	q.Dequeue()

	q.Close()

	if _, exp := q.Dequeue(); exp != -1 {
		t.Fatal("队列已关闭，Dequeue 获取到的过期时间应该是 -1")
	}

	if ele := q.Enqueue(4, now+4); ele != nil {
		t.Fatal("队列已关闭，Enqueue 的返回值应该是 nil")
	}
}
