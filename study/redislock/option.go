package redislock

import "time"

type Option struct {
	Timeout  time.Duration
	Interval time.Duration
	TryTimes int64
}

func defaultOption() *Option {
	return &Option{
		Timeout:  10 * time.Second,
		TryTimes: 20,
		Interval: 50 * time.Millisecond,
	}
}

type optionFn func(cfg *Option)

// WithTimeout 锁过期时间
func WithTimeout(timeout time.Duration) optionFn {
	return func(opt *Option) {
		opt.Timeout = timeout
	}
}

// WithInterval 尝试获取锁间隔
func WithInterval(interval time.Duration) optionFn {
	return func(opt *Option) {
		opt.Interval = interval
	}
}

// WithTryTimes 尝试获取锁次数
func WithTryTimes(times int64) optionFn {
	return func(opt *Option) {
		opt.TryTimes = times
	}
}
