package lru

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	lru := New[string, string](2, func(key string, value string) {
		fmt.Printf("%v:%v deleted\n", key, value)
	})
	ttl := time.Now().Add(time.Minute).UnixNano()
	lru.Add("key1", "1234", ttl)
	lru.Add("key2", "12", ttl)

	fmt.Println(lru.Get("key1"))
	lru.Add("key3", "14", ttl)
	fmt.Println(lru.Get("key2"))
	if v, ok := lru.Get("key1"); !ok || v != "1234" {
		t.Fatalf("cache hit key1=1234 failed")
	}
	if v, ok := lru.Get("key3"); !ok || v != "14" {
		t.Fatalf("cache hit key3=14 failed")
	}
	if _, ok := lru.Get("key2"); ok {
		t.Fatalf("cache miss key2 failed")
	}
}

func TestRemoveoldest(t *testing.T) {
	k1, k2, k3 := "key1", "key2", "k3"
	v1, v2, v3 := "value1", "value2", "v3"
	lru := New[string, string](2, func(key string, value string) {
		fmt.Printf("%v:%v deleted\n", key, value)
	})
	ttl := time.Now().Add(time.Minute).UnixNano()
	lru.Add(k1, v1, ttl)
	lru.Add(k2, v2, ttl)
	lru.Add(k3, v3, ttl)
	fmt.Println(lru.Get("key1"))
	fmt.Println(lru.Get("key2"))
	fmt.Println(lru.Get("k3"))
	if _, ok := lru.Get("key1"); ok || lru.Len() != 2 {
		t.Fatalf("Removeoldest key1 failed")
	}
}

func TestExpires(t *testing.T) {
	k1, k2, k3, k4 := "key1", "key2", "k3", "k4"
	v1, v2, v3, v4 := "value1", "value2", "v3", "v4"
	lru := New[string, string](100, func(key string, value string) {
		fmt.Printf("%v:%v deleted\n", key, value)
	})
	lru.Add(k1, v1, time.Now().Add(1*time.Second).UnixNano())
	lru.Add(k2, v2, time.Now().Add(2*time.Second).UnixNano())
	lru.Add(k3, v3, time.Now().Add(3*time.Second).UnixNano())
	lru.Add(k4, v4, 0)
	time.Sleep(1 * time.Second)
	fmt.Println(lru.Get("key1"))
	time.Sleep(1 * time.Second)
	fmt.Println(lru.Get("key2"))
	time.Sleep(1 * time.Second)
	fmt.Println(lru.Get("k3"))
	fmt.Println(lru.Get("k4"))
	if _, ok := lru.Get("k4"); !ok || lru.Len() != 1 {
		t.Fatalf("Removeoldest k3 failed")
	}
}

func TestOnRemove(t *testing.T) {
	keys := make([]string, 0)
	lru := New[string, string](2, func(key string, value string) {
		keys = append(keys, key)
	})
	ttl := time.Now().Add(time.Minute).UnixNano()
	lru.Add("key1", "123456", ttl)
	lru.Add("k2", "k2", ttl)
	lru.Add("k3", "k3", ttl)
	lru.Add("k4", "k4", ttl)

	expect := []string{"key1", "k2"}
	fmt.Println(keys)
	if !reflect.DeepEqual(expect, keys) {
		t.Fatalf("Call onEvicted failed, expect keys equals to %s", expect)
	}
}
