package limitbucket

import (
	"context"
	"math/rand"
	"time"
)

type Bucket struct {
	cfg *config
	q   *queue
}

func NewBucket(opts ...option) *Bucket {
	cfg := &config{
		Size:     60,
		Interval: 60 * time.Second,
		Strategy: Disposable,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return &Bucket{cfg: cfg, q: &queue{}}
}

func (b *Bucket) Start(ctx context.Context) {
	ctx, _ = context.WithCancel(ctx)
	go b.startGenerater(ctx)
}

func (b *Bucket) startGenerater(ctx context.Context) {
	switch b.cfg.Strategy {
	case Disposable:
		b.startDisposableGenerater(ctx)
	case Average:
		b.startAverageGenerater(ctx)
	}
}

func (b *Bucket) startDisposableGenerater(ctx context.Context) {
	for {
		select {
		case <-time.After(b.cfg.Interval):
			for i := 0; i < b.cfg.Size; i++ {
				token := b.generateToken()
				if b.q.Len() < b.cfg.Size {
					b.q.Push(token)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}
func (b *Bucket) startAverageGenerater(ctx context.Context) {
	for {
		select {
		case <-time.After(b.cfg.Interval / time.Duration(b.cfg.Size)):
			token := b.generateToken()
			if b.q.Len() < b.cfg.Size {
				b.q.Push(token)
			}
		case <-ctx.Done():
			return
		}
	}
}

func (b *Bucket) GetToken() string {
	token := b.q.Pop()
	if token == nil {
		return ""
	}

	return token.(string)
}

func (b *Bucket) generateToken() string {
	return RandString(32)
}

func RandString(len int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		b := r.Intn(26) + 65
		bytes[i] = byte(b)
	}
	return string(bytes)
}
