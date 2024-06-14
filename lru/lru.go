// Package lru implements an LRU cache.
package lru

import (
	"kunCache/list"
)

// Cache is an LRU cache. It is not safe for concurrent access.
type Cache[K comparable, V any] struct {
	// maxEntries is the maximum number of cache entries before
	// an node is evicted. Zero means no limit.
	maxEntries int64

	// onEvicted optionally specifies a callback function to be
	// executed when an entry is purged from the cache.
	onEvicted func(key K, value V)

	ll    *list.List[K, V]
	cache map[K]*list.Node[K, V]
}

// New creates a new Cache.
// If maxEntries is zero, the cache has no limit and it's assumed
// that eviction is done by the caller.
func New[K comparable, V any](maxEntries int64, onEvicted func(key K, value V)) *Cache[K, V] {
	return &Cache[K, V]{
		maxEntries: maxEntries,
		ll:         list.NewList[K, V](),
		cache:      make(map[K]*list.Node[K, V]),
		onEvicted:  onEvicted,
	}
}

// Add adds a value to the cache.
func (c *Cache[K, V]) Add(key K, value V, expires int64) {
	if node, ok := c.cache[key]; ok {
		if c.onEvicted != nil {
			c.onEvicted(node.Key(), node.Value())
		}
		c.ll.MoveToFront(node)
		node.SetExpires(expires)
		node.SetValue(value)
		return
	}
	node := c.ll.Insert(key, value, expires)
	c.cache[key] = node
	if c.maxEntries != 0 && c.Len() > c.maxEntries {
		c.RemoveOldest()
	}
}

// Get looks up a key's value from the cache.
func (c *Cache[K, V]) Get(key K) (value V, ok bool) {
	if node, hit := c.cache[key]; hit {
		// If the value has expired, remove it from the cache
		if node.Expires().UnixNano() != 0 && node.Expired() {
			c.removeElement(node)
			return
		}
		c.ll.MoveToFront(node)
		return node.Value(), true
	}
	return
}

// Remove removes the provided key from the cache.
func (c *Cache[K, V]) Remove(key K) {
	if node, hit := c.cache[key]; hit {
		c.removeElement(node)
	}
}

// RemoveOldest removes the oldest node from the cache.
func (c *Cache[K, V]) RemoveOldest() {
	node := c.ll.Tail.Prev
	if node != c.ll.Head {
		c.removeElement(node)
	}
}

func (c *Cache[K, V]) removeElement(node *list.Node[K, V]) {
	c.ll.Remove(node)
	delete(c.cache, node.Key())
	if c.onEvicted != nil {
		c.onEvicted(node.Key(), node.Value())
	}
}

// Len returns the number of items in the cache.
func (c *Cache[K, V]) Len() int64 {
	return c.ll.Len()
}

// Clear purges all stored items from the cache.
func (c *Cache[K, V]) Clear() {
	if c.onEvicted != nil {
		for _, node := range c.cache {
			c.onEvicted(node.Key(), node.Value())
		}
	}
	c.ll = nil
	c.cache = nil
}
