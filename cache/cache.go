package cache

import (
	"kunCache/lru"
	"sync"
)

type Cache[K comparable, V any] struct {
	mu  sync.RWMutex //lru 读写都会修改内容
	lru *lru.Cache[K, V]
}

func New[K comparable, V any](maxEntries int64, onEvicted func(key K, value V)) *Cache[K, V] {
	return &Cache[K, V]{
		lru: lru.New[K, V](maxEntries, onEvicted),
	}
}
func (c *Cache[K, V]) Add(key K, value V, expires int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lru.Add(key, value, expires)
}

func (c *Cache[K, V]) Get(key K) (value V, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lru.Get(key)
}
