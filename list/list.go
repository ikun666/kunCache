package list

// 头尾节点不实际存储 head---tail
type List[K comparable, V any] struct {
	Head *Node[K, V]
	Tail *Node[K, V]
	len  int64
}

func NewList[K comparable, V any]() *List[K, V] {
	head := &Node[K, V]{}
	tail := &Node[K, V]{}
	head.Next = tail
	tail.Prev = head
	return &List[K, V]{
		Head: head,
		Tail: tail,
		len:  0,
	}
}
func (l *List[K, V]) Len() int64 {
	return l.len
}
func (l *List[K, V]) Remove(node *Node[K, V]) {
	node.Prev.Next = node.Next
	node.Next.Prev = node.Prev
	l.len--
}

func (l *List[K, V]) MoveToFront(node *Node[K, V]) {
	l.Remove(node)
	l.nodeToFront(node)
}

func (l *List[K, V]) Insert(key K, value V, expires int64) *Node[K, V] {
	node := newNode[K, V](key, value, expires)
	l.nodeToFront(node)
	return node
}

func (l *List[K, V]) nodeToFront(node *Node[K, V]) {
	node.Next = l.Head.Next
	l.Head.Next.Prev = node
	l.Head.Next = node
	node.Prev = l.Head
	l.len++
}
