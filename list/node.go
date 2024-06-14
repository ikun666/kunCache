package list

import (
	"fmt"
	"time"
)

type Node[K comparable, V any] struct {
	key     K
	value   V
	expires int64
	Next    *Node[K, V]
	Prev    *Node[K, V]
}

func newNode[K comparable, V any](key K, value V, expires int64) *Node[K, V] {
	return &Node[K, V]{
		key:     key,
		value:   value,
		expires: expires,
	}
}

func (n *Node[K, V]) Key() K {
	return n.key
}

func (n *Node[K, V]) Value() V {
	return n.value
}

func (n *Node[K, V]) SetValue(value V) {
	n.value = value
}
func (n *Node[K, V]) Expired() bool {
	return n.expires < time.Now().UnixNano()
}

func (n *Node[K, V]) TTL() time.Duration {
	return time.Nanosecond * time.Duration(n.expires-time.Now().UnixNano())
}

func (n *Node[K, V]) Expires() time.Time {
	return time.Unix(0, n.expires)
}
func (n *Node[K, V]) SetExpires(expires int64) {
	n.expires = expires
}
func (n *Node[K, V]) Extend(duration time.Duration) {
	n.expires = time.Now().Add(duration).UnixNano()
}

// String returns a string representation of the Node. This includes the default string
// representation of its Value(), as implemented by fmt.Sprintf with "%v", but the exact
// format of the string should not be relied on; it is provided only for debugging
// purposes, and because otherwise including an Node in a call to fmt.Printf or
// fmt.Sprintf expression could cause fields of the Node to be read in a non-thread-safe
// way.
func (n *Node[K, V]) String() string {
	return fmt.Sprintf("Node(key:%v value:%v expires:%v)", n.Key(), n.Value(), n.Expires())
}
