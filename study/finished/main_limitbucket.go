package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"winse.com/study/limitbucket"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	b := limitbucket.NewBucket(limitbucket.BucketSize(100), limitbucket.Interval(1*time.Second), limitbucket.Strategy(limitbucket.Disposable))
	b.Start(ctx)

	<-time.After(1 * time.Second)

	count := 0
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(ctx context.Context, no int) {
			defer wg.Done()
			for {
				select {
				case <-time.After(50 * time.Millisecond):
					t := b.GetToken()
					if t != "" {
						count++
					}
					fmt.Printf("[%d]: %s \n", no, t)
				case <-ctx.Done():
					return
				}
			}
		}(ctx, i)
	}

	<-time.After(2 * time.Second)
	cancel()

	wg.Wait()
	fmt.Println(count)
}
