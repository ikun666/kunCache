package list

import (
	"fmt"
	"testing"
	"time"
)

func listPrint[K comparable, V any](l *List[K, V]) {
	p := l.Head.Next
	fmt.Printf("len:%v\n", l.Len())
	for p != l.Tail {
		fmt.Printf("%v\n", p)
		p = p.Next
	}
	fmt.Println()
}
func Test_List_Insert(t *testing.T) {
	l := NewList[string, int]()
	listPrint(l)
	ttl := time.Now().Add(time.Minute).UnixNano()
	l.Insert("1", 1, ttl)
	listPrint(l)

	l.Insert("2", 2, ttl)
	listPrint(l)

	l.Insert("3", 3, ttl)
	listPrint(l)
}

func Test_List_Remove(t *testing.T) {
	l := NewList[string, int]()
	listPrint(l)
	ttl := time.Now().Add(time.Minute).UnixNano()
	node := l.Insert("1", 1, ttl)
	l.Remove(node)
	listPrint(l)

	n5 := l.Insert("5", 5, ttl)
	n4 := l.Insert("4", 4, ttl)
	n3 := l.Insert("3", 3, ttl)
	n2 := l.Insert("2", 2, ttl)
	n1 := l.Insert("1", 1, ttl)

	l.Remove(n5)
	listPrint(l)

	l.Remove(n1)
	listPrint(l)

	l.Remove(n3)
	listPrint(l)

	l.Remove(n2)
	listPrint(l)

	l.Remove(n4)
	listPrint(l)
}

func Test_List_MoveToFront(t *testing.T) {
	l := NewList[string, int]()
	ttl := time.Now().Add(time.Minute).UnixNano()
	n1 := l.Insert("1", 1, ttl)
	l.MoveToFront(n1)
	listPrint(l)

	n2 := l.Insert("2", 2, ttl)
	l.MoveToFront(n1)
	listPrint(l)
	l.MoveToFront(n2)
	listPrint(l)
}
