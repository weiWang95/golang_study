package limitbucket

import "time"

type strategy int

const (
	Disposable strategy = iota
	Average
)

type config struct {
	Size     int
	Interval time.Duration
	Strategy strategy
}

type option func(cfg *config)

func BucketSize(size int) option {
	return func(cfg *config) {
		cfg.Size = size
	}
}

func Interval(d time.Duration) option {
	return func(cfg *config) {
		cfg.Interval = d
	}
}

func Strategy(s strategy) option {
	return func(cfg *config) {
		cfg.Strategy = s
	}
}
