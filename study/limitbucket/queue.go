package limitbucket

import "sync"

type queue struct {
	mux  sync.RWMutex
	data []interface{}
}

func NewQueue() *queue {
	return &queue{data: []interface{}{}}
}

func (q *queue) Push(item interface{}) {
	q.mux.Lock()
	defer q.mux.Unlock()

	q.data = append(q.data, item)
}

func (q *queue) Pop() interface{} {
	q.mux.Lock()
	defer q.mux.Unlock()

	if len(q.data) == 0 {
		return nil
	}

	item := q.data[0]
	q.data = q.data[1:]
	return item
}

func (q *queue) Len() int {
	q.mux.RLock()
	defer q.mux.RUnlock()

	return len(q.data)
}

func (q *queue) IsEmpty() bool {
	return q.Len() == 0
}
