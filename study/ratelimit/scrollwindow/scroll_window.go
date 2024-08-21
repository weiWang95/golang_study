package scrollwindow

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type ScrollWindow struct {
	limit    int64
	duration time.Duration

	step  time.Duration
	count int

	totalReq int64

	list    *SubWindowList
	closeCh chan struct{}
}

func NewScrollWindow(limit int64, duration time.Duration, step time.Duration) *ScrollWindow {
	w := new(ScrollWindow)
	w.limit = limit
	w.duration = duration
	w.step = step
	w.count = int(duration / step)
	w.list = &SubWindowList{}
	return w
}

func (w *ScrollWindow) Start() {
	w.closeCh = w.generateWindow()
}

func (w *ScrollWindow) Stop() {
	if w.closeCh == nil {
		return
	}

	w.closeCh <- struct{}{}
	w.closeCh = nil
}

func (w *ScrollWindow) Check() bool {
	if w.totalReq >= w.limit {
		return false
	}

	atomic.AddInt64(&w.list.First().count, 1)
	atomic.AddInt64(&w.totalReq, 1)
	return true
}

func (w *ScrollWindow) generateWindow() chan struct{} {
	now := time.Now().UnixNano()
	w.list.Push(NewSubWindow(now, now+w.step.Nanoseconds()))

	close := make(chan struct{})
	go func() {
		t := time.NewTicker(w.step)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				now := time.Now().UnixNano()
				w.list.Push(NewSubWindow(now, now+w.step.Nanoseconds()))
				if w.list.Len() >= w.count {
					n := w.list.Shift()
					fmt.Printf("shift: [%d,%d] %d - %d\n", n.start, n.end, w.totalReq, n.count)
					atomic.AddInt64(&w.totalReq, -n.count)
				}
			case <-close:
				return
			}
		}
	}()
	return close
}

func (w *ScrollWindow) Inspect() {
	w.list.Each(func(n *SubWindow) bool {
		fmt.Printf("[%d,%d] %d\n", n.start, n.end, n.count)
		return true
	})
}

type SubWindowList struct {
	first *SubWindow
	last  *SubWindow

	len int
	mux sync.RWMutex
}

func (l *SubWindowList) Len() int {
	return l.len
}

func (l *SubWindowList) Each(fn func(item *SubWindow) bool) {
	l.mux.RLock()
	defer l.mux.RUnlock()

	if l.first == nil {
		return
	}

	cur := l.first
	for {
		if !fn(cur) {
			break
		}

		if cur.next == nil {
			break
		}

		cur = cur.next
	}
}

func (l *SubWindowList) First() *SubWindow {
	l.mux.RLock()
	defer l.mux.RUnlock()
	return l.first
}

func (l *SubWindowList) Last() *SubWindow {
	l.mux.RLock()
	defer l.mux.RUnlock()
	return l.last
}

func (l *SubWindowList) Push(item *SubWindow) {
	l.mux.Lock()
	defer l.mux.Unlock()

	if l.first == nil {
		l.first = item
		l.last = item
		return
	}

	item.next = l.first
	l.first.prev = item
	l.first = item
	l.len += 1
}

func (l *SubWindowList) Shift() *SubWindow {
	l.mux.Lock()
	defer l.mux.Unlock()

	if l.last == nil {
		return nil
	}
	last := l.last

	if l.last.prev == nil {
		l.first = nil
		l.last = nil
		return last
	}
	l.last = l.last.prev
	l.last.next = nil

	last.prev = nil
	l.len -= 1
	return last
}

type SubWindow struct {
	prev *SubWindow
	next *SubWindow

	count int64
	start int64
	end   int64
}

func NewSubWindow(start, end int64) *SubWindow {
	w := new(SubWindow)
	w.start = start
	w.end = end
	return w
}

func (w *SubWindow) Prev() *SubWindow {
	return w.prev
}

func (w *SubWindow) SetPrev(item *SubWindow) {
	w.prev = item
}

func (w *SubWindow) Next() *SubWindow {
	return w.next
}

func (w *SubWindow) SetNext(item *SubWindow) {
	w.next = item
}
