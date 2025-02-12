package ringqueue

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestRingQueue(t *testing.T) {
	q := NewRingQueue[int](10)

	w := sync.WaitGroup{}
	ch, sig := make(chan struct{}), make(chan struct{})

	for i := range 2 {
		w.Add(1)

		go func(idx int) {
			defer w.Done()
			<-sig
			for j := range 4 {
				d := j*2 + idx
				if q.Enqueue(d) {
					fmt.Printf("%v Enqueue %v\n", idx, d)
				}

				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	go func() {
		close(sig)
		for {
			select {
			case <-ch:
				return
			default:

				if r, b := q.Dequeue(); b {
					fmt.Printf("  Dequeue %v\n", r)
				}
			}
		}
	}()

	w.Wait()
	close(ch)

	t.Fail()
}
