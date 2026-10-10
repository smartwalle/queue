package delayqueue_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/smartwalle/queue"
	"github.com/smartwalle/queue/delayqueue"
)

// item 是带绝对到期时间的元素。
type item struct {
	name     string
	deadline time.Time
}

// newDelayQueue 按元素自带的绝对到期时间创建队列。
func newDelayQueue() *delayqueue.Queue[item] {
	return delayqueue.New[item](func(v item) time.Time {
		return v.deadline
	})
}

func BenchmarkDelayQueue_Enqueue(b *testing.B) {
	var q = newDelayQueue()
	var deadline = time.Now().Add(-time.Hour)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := q.Enqueue(item{deadline: deadline}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDelayQueue_EnqueueDequeue(b *testing.B) {
	var q = newDelayQueue()
	var deadline = time.Now().Add(-time.Hour)

	var done = make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, err := q.Dequeue(); err != nil {
				return
			}
		}
	}()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := q.Enqueue(item{deadline: deadline}); err != nil {
			b.Fatal(err)
		}
	}
	q.Close()
	<-done
}

func BenchmarkDelayQueue_Dequeue(b *testing.B) {
	var q = newDelayQueue()
	var deadline = time.Now().Add(-time.Hour)
	for i := 0; i < b.N; i++ {
		if err := q.Enqueue(item{deadline: deadline}); err != nil {
			b.Fatal(err)
		}
	}
	// 关闭后 Dequeue 不再阻塞。
	q.Close()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := q.Dequeue(); err != nil {
			b.Fatal("Dequeue 异常", err)
		}
	}
}

func TestDelayQueue_Len(t *testing.T) {
	var q = newDelayQueue()

	if q.Len() != 0 {
		t.Fatalf("新建队列长度 %d，期望 0", q.Len())
	}

	// 尚未到期的元素也计入 Len。
	for i := 0; i < 3; i++ {
		if err := q.Enqueue(item{name: "a", deadline: time.Now().Add(time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}

	if q.Len() != 3 {
		t.Fatalf("队列长度 %d，期望 3", q.Len())
	}
}

func TestDelayQueue_DequeueOrder(t *testing.T) {
	var q = newDelayQueue()

	// 全部已到期，按到期时间升序出队。
	var list = []item{
		{name: "c", deadline: time.Now().Add(-30 * time.Millisecond)},
		{name: "a", deadline: time.Now().Add(-90 * time.Millisecond)},
		{name: "d", deadline: time.Now().Add(-10 * time.Millisecond)},
		{name: "b", deadline: time.Now().Add(-60 * time.Millisecond)},
	}
	for _, it := range list {
		if err := q.Enqueue(it); err != nil {
			t.Fatal(err)
		}
	}
	q.Close()

	var expect = []string{"a", "b", "c", "d"}
	for _, want := range expect {
		var got, err = q.Dequeue()
		if err != nil {
			t.Fatalf("出队 %s 失败: %v", want, err)
		}
		if got.name != want {
			t.Fatalf("出队顺序与预期不符，期望 %s，实际 %s", want, got.name)
		}
	}

	if q.Len() != 0 {
		t.Fatalf("队列长度 %d，期望 0", q.Len())
	}
}

func TestDelayQueue_TryDequeue(t *testing.T) {
	var q = newDelayQueue()

	if _, err := q.TryDequeue(); !errors.Is(err, queue.ErrEmpty) {
		t.Fatalf("空队列 TryDequeue 应返回 ErrEmpty，实际 %v", err)
	}

	if err := q.Enqueue(item{name: "a", deadline: time.Now().Add(-time.Millisecond)}); err != nil {
		t.Fatal(err)
	}

	// 已到期，不阻塞。
	var got, err = q.TryDequeue()
	if err != nil {
		t.Fatal(err)
	}
	if got.name != "a" {
		t.Fatalf("出队元素 %s，期望 a", got.name)
	}

	// 未到期的元素同样返回 ErrEmpty，且不会出队。
	if err := q.Enqueue(item{name: "b", deadline: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if _, err := q.TryDequeue(); !errors.Is(err, queue.ErrEmpty) {
		t.Fatalf("元素未到期时 TryDequeue 应返回 ErrEmpty，实际 %v", err)
	}
	if q.Len() != 1 {
		t.Fatalf("队列长度 %d，期望 1，未到期元素不应被取出", q.Len())
	}

	// 关闭后元素仍在，未到期时仍返回 ErrEmpty。
	q.Close()
	if _, err := q.TryDequeue(); !errors.Is(err, queue.ErrEmpty) {
		t.Fatalf("关闭后元素未到期时 TryDequeue 应返回 ErrEmpty，实际 %v", err)
	}

	// 空队列且已关闭返回 ErrClosed。
	var empty = newDelayQueue()
	empty.Close()
	if _, err := empty.TryDequeue(); !errors.Is(err, queue.ErrClosed) {
		t.Fatalf("关闭且排空后 TryDequeue 应返回 ErrClosed，实际 %v", err)
	}
}

func TestDelayQueue_DequeueWait(t *testing.T) {
	var q = newDelayQueue()

	if err := q.Enqueue(item{name: "a", deadline: time.Now().Add(200 * time.Millisecond)}); err != nil {
		t.Fatal(err)
	}

	var (
		got      item
		err      error
		elapsed  time.Duration
		returned = make(chan struct{})
	)
	var start = time.Now()
	go func() {
		defer close(returned)
		got, err = q.Dequeue()
		elapsed = time.Since(start)
	}()

	select {
	case <-returned:
		t.Fatal("元素未到期时 Dequeue 不应立即返回")
	case <-time.After(100 * time.Millisecond):
	}

	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("到期后 Dequeue 未返回")
	}

	if err != nil {
		t.Fatal(err)
	}
	if got.name != "a" {
		t.Fatalf("出队元素 %s，期望 a", got.name)
	}
	if elapsed < 150*time.Millisecond {
		t.Fatalf("Dequeue 耗时 %v，短于到期时间", elapsed)
	}
}

func TestDelayQueue_CloseWaitDequeue(t *testing.T) {
	var q = newDelayQueue()

	// 空队列上 Dequeue 阻塞，Close 后应返回 ErrClosed。
	var returned = make(chan struct{})
	go func() {
		defer close(returned)
		if _, err := q.Dequeue(); !errors.Is(err, queue.ErrClosed) {
			t.Errorf("期望 ErrClosed，实际 %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	q.Close()

	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("Close 之后阻塞的 Dequeue 未被唤醒")
	}
}

func TestDelayQueue_Close(t *testing.T) {
	var q = newDelayQueue()

	if q.Closed() {
		t.Fatal("新建的队列不应是关闭状态")
	}

	if err := q.Enqueue(item{name: "a", deadline: time.Now().Add(-time.Millisecond)}); err != nil {
		t.Fatal(err)
	}

	q.Close()
	// 重复关闭不应 panic。
	q.Close()

	if !q.Closed() {
		t.Fatal("Close 之后 Closed 应返回 true")
	}

	if err := q.Enqueue(item{name: "b", deadline: time.Now()}); !errors.Is(err, queue.ErrClosed) {
		t.Fatalf("关闭后 Enqueue 应返回 ErrClosed，实际 %v", err)
	}

	// 关闭后仍可取出存量元素。
	if got, err := q.Dequeue(); err != nil || got.name != "a" {
		t.Fatalf("关闭后 Dequeue 期望 (a, nil)，实际 (%v, %v)", got.name, err)
	}

	// 关闭且排空后返回 ErrClosed。
	if _, err := q.Dequeue(); !errors.Is(err, queue.ErrClosed) {
		t.Fatalf("关闭且排空后 Dequeue 应返回 ErrClosed，实际 %v", err)
	}
}

func TestDelayQueue_Concurrent(t *testing.T) {
	var (
		producers = 4
		perTask   = 200
		consumers = 3
		total     = producers * perTask
	)

	// 元素立即到期，出队顺序不确定，只校验总数不重不漏。
	var q = newDelayQueue()

	var received atomic.Int64

	var cwg sync.WaitGroup
	cwg.Add(consumers)
	for i := 0; i < consumers; i++ {
		go func() {
			defer cwg.Done()
			for {
				if _, err := q.Dequeue(); err != nil {
					return
				}
				received.Add(1)
			}
		}()
	}

	var pwg sync.WaitGroup
	pwg.Add(producers)
	for p := 0; p < producers; p++ {
		go func() {
			defer pwg.Done()
			for j := 0; j < perTask; j++ {
				if err := q.Enqueue(item{name: "x", deadline: time.Now()}); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}

	pwg.Wait()
	q.Close()
	cwg.Wait()

	if got := received.Load(); got != int64(total) {
		t.Fatalf("出队 %d 个元素，期望 %d", got, total)
	}
	if q.Len() != 0 {
		t.Fatalf("队列长度 %d，期望 0", q.Len())
	}
}
