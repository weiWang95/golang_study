package redislock

import "errors"

var (
	ErrTryTimesOver = errors.New("lock try times over")
	ErrLockFail     = errors.New("lock fail")
)
