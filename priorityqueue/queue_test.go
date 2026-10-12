package priorityqueue_test

import (
	"errors"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/smartwalle/queue"
	"github.com/smartwalle/queue/priorityqueue"
)

// newPriorityQueue 按元素值比较创建队列：数值小的优先出队。
func newPriorityQueue() *priorityqueue.Queue[int] {
	return priorityqueue.New[int](0, func(a, b int) bool {
		return a < b
	})
}

func BenchmarkPriorityQueue_Enqueue(b *testing.B) {
	var priorities = make([]int64, b.N)
	var r = rand.New(rand.NewSource(42))
	for i := range priorities {
		priorities[i] = r.Int63()
	}

	var q = priorityqueue.New[int](0, func(a, b int) bool {
		return priorities[a] < priorities[b]
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := q.Enqueue(i); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPriorityQueue_Dequeue(b *testing.B) {
	var priorities = make([]int64, b.N)
	var r = rand.New(rand.NewSource(42))
	for i := range priorities {
		priorities[i] = r.Int63()
	}

	var q = priorityqueue.New[int](0, func(a, b int) bool {
		return priorities[a] < priorities[b]
	})
	for i := 0; i < b.N; i++ {
		if err := q.Enqueue(i); err != nil {
			b.Fatal(err)
		}
	}
	// 关闭后 Dequeue 不再阻塞。
	q.Close()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := q.Dequeue(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPriorityQueue_EnqueueDequeue(b *testing.B) {
	var q = newPriorityQueue()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := q.Enqueue(1); err != nil {
			b.Fatal(err)
		}
		if _, err := q.Dequeue(); err != nil {
			b.Fatal(err)
		}
	}
}

func TestPriorityQueue_Dequeue(t *testing.T) {
	var q = newPriorityQueue()

	var list = []int{6, 1, 8, 2, 9, 3, 5, 4, 7, 0}
	for _, item := range list {
		if err := q.Enqueue(item); err != nil {
			t.Fatal(err)
		}
	}

	if q.Len() != len(list) {
		t.Fatalf("队列长度 %d，期望 %d", q.Len(), len(list))
	}

	// 关闭后仍可取出剩余元素。
	q.Close()

	var nList = make([]int, 0, len(list))
	for {
		var item, err = q.Dequeue()
		if err != nil {
			break
		}
		nList = append(nList, item)
	}

	if len(nList) != len(list) {
		t.Fatalf("出队 %d 个元素，期望 %d", len(nList), len(list))
	}

	sort.Ints(list)

	for idx, item := range nList {
		if item != list[idx] {
			t.Fatalf("出队顺序与预期不符，期望 %d，实际 %d", list[idx], item)
		}
	}
}

func TestPriorityQueue_SamePriority(t *testing.T) {
	// 比较函数恒为 false 时，所有元素优先级相同，出队顺序不确定，
	// 这里只校验全部元素都能出队、一个不丢。
	var q = priorityqueue.New[int](0, func(a, b int) bool {
		return false
	})

	var total = 1000
	var counts = make([]int32, total)
	for i := 0; i < total; i++ {
		if err := q.Enqueue(i); err != nil {
			t.Fatal(err)
		}
	}
	if q.Len() != total {
		t.Fatalf("队列长度 %d，期望 %d", q.Len(), total)
	}
	q.Close()

	for i := 0; i < total; i++ {
		var item, err = q.Dequeue()
		if err != nil {
			t.Fatalf("出队第 %d 个元素失败: %v", i, err)
		}
		counts[item]++
	}

	for i, count := range counts {
		if count != 1 {
			t.Fatalf("元素 %d 被出队 %d 次，期望 1 次", i, count)
		}
	}
}

func TestPriorityQueue_TryDequeue(t *testing.T) {
	var q = newPriorityQueue()

	if _, err := q.TryDequeue(); !errors.Is(err, queue.ErrEmpty) {
		t.Fatalf("空队列 TryDequeue 应返回 ErrEmpty，实际 %v", err)
	}

	if err := q.Enqueue(3); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(1); err != nil {
		t.Fatal(err)
	}

	// 不会阻塞，按优先级出队。
	if item, err := q.TryDequeue(); err != nil || item != 1 {
		t.Fatalf("TryDequeue 期望 (1, nil)，实际 (%d, %v)", item, err)
	}
	if item, err := q.TryDequeue(); err != nil || item != 3 {
		t.Fatalf("TryDequeue 期望 (3, nil)，实际 (%d, %v)", item, err)
	}
	if q.Len() != 0 {
		t.Fatalf("队列长度 %d，期望 0", q.Len())
	}
}

func TestPriorityQueue_Close(t *testing.T) {
	var q = newPriorityQueue()

	if q.Closed() {
		t.Fatal("新建的队列不应是关闭状态")
	}

	if err := q.Enqueue(1); err != nil {
		t.Fatal(err)
	}

	q.Close()
	// 重复关闭不应 panic。
	q.Close()

	if !q.Closed() {
		t.Fatal("Close 之后 Closed 应返回 true")
	}

	if err := q.Enqueue(2); !errors.Is(err, queue.ErrClosed) {
		t.Fatalf("关闭后 Enqueue 应返回 ErrClosed，实际 %v", err)
	}

	// 关闭后仍可取出存量元素。
	if item, err := q.Dequeue(); err != nil || item != 1 {
		t.Fatalf("关闭后 Dequeue 期望 (1, nil)，实际 (%d, %v)", item, err)
	}

	// 关闭且排空后返回 ErrClosed。
	if _, err := q.Dequeue(); !errors.Is(err, queue.ErrClosed) {
		t.Fatalf("关闭且排空后 Dequeue 应返回 ErrClosed，实际 %v", err)
	}
	if _, err := q.TryDequeue(); !errors.Is(err, queue.ErrClosed) {
		t.Fatalf("关闭且排空后 TryDequeue 应返回 ErrClosed，实际 %v", err)
	}
}

func TestPriorityQueue_DequeueWait(t *testing.T) {
	var q = newPriorityQueue()

	var got int
	var returned = make(chan struct{})
	go func() {
		defer close(returned)
		item, err := q.Dequeue()
		if err != nil {
			t.Error(err)
			return
		}
		got = item
	}()

	select {
	case <-returned:
		t.Fatal("空队列时 Dequeue 不应立即返回")
	case <-time.After(100 * time.Millisecond):
	}

	if err := q.Enqueue(7); err != nil {
		t.Fatal(err)
	}

	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("入队后 Dequeue 未被唤醒")
	}

	if got != 7 {
		t.Fatalf("Dequeue 返回 %d，期望 7", got)
	}
}

func TestPriorityQueue_CloseWaitDequeue(t *testing.T) {
	var q = newPriorityQueue()

	var returned = make(chan struct{})
	go func() {
		defer close(returned)
		if _, err := q.Dequeue(); !errors.Is(err, queue.ErrClosed) {
			t.Errorf("关闭后空队列 Dequeue 应返回 ErrClosed，实际 %v", err)
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

func TestPriorityQueue_Concurrent(t *testing.T) {
	var (
		producers = 8
		perTask   = 1000
		consumers = 4
		total     = producers * perTask
	)

	var q = newPriorityQueue()

	// 每个元素的值唯一，counts 记录其被出队的次数，应恰好为 1。
	var counts = make([]int32, total)

	var cwg sync.WaitGroup
	cwg.Add(consumers)
	for i := 0; i < consumers; i++ {
		go func() {
			defer cwg.Done()
			for {
				var item, err = q.Dequeue()
				if err != nil {
					return
				}
				atomic.AddInt32(&counts[item], 1)
			}
		}()
	}

	var pwg sync.WaitGroup
	pwg.Add(producers)
	for p := 0; p < producers; p++ {
		p := p
		go func() {
			defer pwg.Done()
			for j := 0; j < perTask; j++ {
				if err := q.Enqueue(p*perTask + j); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}

	pwg.Wait()
	q.Close()
	cwg.Wait()

	for i, count := range counts {
		if count != 1 {
			t.Fatalf("元素 %d 被出队 %d 次，期望 1 次", i, count)
		}
	}
}
