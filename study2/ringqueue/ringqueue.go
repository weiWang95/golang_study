package ringqueue

import (
	"sync/atomic"
)

type RingQueue[T any] struct {
	buf  []T
	head int64
	tail int64
	size int64
}

func NewRingQueue[T any](size int64) *RingQueue[T] {
	return &RingQueue[T]{
		buf:  make([]T, size),
		size: size,
	}
}

func (r *RingQueue[T]) Enqueue(t T) bool {
	for {
		tail := atomic.LoadInt64(&r.tail)
		nextTail := (tail + 1) % r.size

		// 已满
		if nextTail == atomic.LoadInt64(&r.head) {
			return false
		}

		if atomic.CompareAndSwapInt64(&r.tail, tail, nextTail) {
			r.buf[tail] = t
			return true
		}

		// time.Sleep(time.Millisecond)
	}
}

func (r *RingQueue[T]) Dequeue() (T, bool) {
	for {
		head := atomic.LoadInt64(&r.head)
		if head == atomic.LoadInt64(&r.tail) {
			return r.buf[head], false
		}

		nextHead := (head + 1) % r.size

		d := r.buf[head]
		if atomic.CompareAndSwapInt64(&r.head, head, nextHead) {
			return d, true
		}

		// time.Sleep(time.Millisecond)
	}
}
