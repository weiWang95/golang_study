package lru

type LRU[K comparable, V any] struct {
	cache map[K]*node[K, V]
	head  *node[K, V]
	tail  *node[K, V]
	cap   int
}

func NewLRU[K comparable, V any](cap int) *LRU[K, V] {
	return &LRU[K, V]{
		cache: make(map[K]*node[K, V], cap),
		cap:   cap,
	}
}

func (lru *LRU[K, V]) Add(key K, value V) {
	if lru.cap == 0 {
		return
	}

	if _, ok := lru.cache[key]; ok {
		return
	}

	node := &node[K, V]{
		key:   key,
		value: value,
	}

	if lru.head == nil {
		lru.head = node
		lru.tail = node
	} else {
		lru.head.prev = node
		node.next = lru.head
		lru.head = node
	}

	lru.cache[key] = node

	if len(lru.cache) > lru.cap {
		delete(lru.cache, lru.tail.key)
		lru.tail = lru.tail.prev
		lru.tail.next = nil
	}
}

func (lru *LRU[K, V]) Get(key K) *V {
	node, ok := lru.cache[key]
	if !ok {
		return nil
	}

	lru.MoveToHead(key)

	return &node.value
}

func (lru *LRU[K, V]) MoveToHead(key K) {
	node, ok := lru.cache[key]
	if !ok {
		return
	}

	if node == lru.head {
		return
	}

	if node == lru.tail {
		lru.tail = node.prev
		lru.tail.next = nil
	} else {
		node.prev.next = node.next
		node.next.prev = node.prev
	}

	node.prev = nil
	node.next = lru.head
	lru.head.prev = node
	lru.head = node
}

type node[K comparable, V any] struct {
	key   K
	value V
	prev  *node[K, V]
	next  *node[K, V]
}
