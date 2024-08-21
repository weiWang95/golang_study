package redislock

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type ILocker interface {
	Lock(ctx context.Context, key string) error
	Unlock(ctx context.Context, key string) error
}

type locker struct {
	rds *redis.Client
	opt *Option
}

var _ ILocker = (*locker)(nil)

func NewLocker(rds *redis.Client, fns ...optionFn) ILocker {
	opt := defaultOption()
	for _, fn := range fns {
		fn(opt)
	}

	return &locker{
		rds: rds,
		opt: opt,
	}
}

func (l *locker) Lock(ctx context.Context, key string) error {
	var times int64
	for {
		if err := l.tryLock(ctx, key); err != nil {
			if err != ErrLockFail {
				return err
			}
			times += 1
		} else {
			return nil
		}

		if times >= l.opt.TryTimes {
			return ErrTryTimesOver
		}

		time.Sleep(l.opt.Interval)
	}
}

func (l *locker) tryLock(ctx context.Context, key string) error {
	b, err := l.rds.SetNX(ctx, key, 1, l.opt.Timeout).Result()
	if err != nil {
		return err
	}
	if !b {
		return ErrLockFail
	}

	return nil
}

func (l *locker) Unlock(ctx context.Context, key string) error {
	return l.rds.Del(ctx, key).Err()
}
