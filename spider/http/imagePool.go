package http

import (
	"sync"
)

type ImagePool struct {
	sync.RWMutex
	m map[string]int
}

func (pool *ImagePool) Exist(uid string) int {
	pool.RLock()
	defer pool.RUnlock()

	return pool.m[uid]
}

func (pool *ImagePool) Push(uid string, quality int) {
	pool.Lock()
	defer pool.Unlock()

	pool.m[uid] = quality
}

func NewImagePool() *ImagePool {
	return &ImagePool{m: make(map[string]int)}
}