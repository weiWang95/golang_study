package main

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"winse.com/study/ratelimit/scrollwindow"
)

func main() {
	w := scrollwindow.NewScrollWindow(100, time.Second, 200*time.Millisecond)
	w.Start()
	defer w.Stop()

	var wg sync.WaitGroup
	var success, fail int64
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 40; j++ {
				if w.Check() {
					atomic.AddInt64(&success, 1)
				} else {
					atomic.AddInt64(&fail, 1)
				}
				// fmt.Printf("%d %v\n", i, w.Check())
				time.Sleep(100*time.Millisecond + time.Duration(rand.Int63n(50))*time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	fmt.Printf("success:%d fail:%d\n", success, fail)

	w.Inspect()
}
