package singleflight

import "sync"

type call[V any] struct {
	wg  sync.WaitGroup
	val V
	err error
}

type Group[K comparable, V any] struct {
	mu sync.Mutex // protects m
	m  map[K]*call[V]
}

// 包装函数，多次请求执行一次
func (g *Group[K, V]) Do(key K, fn func() (V, error)) (V, error) {
	g.mu.Lock()
	//延迟初始化
	if g.m == nil {
		g.m = make(map[K]*call[V])
	}
	//如果请求存在，等待请求结果
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	//没有加入map，后续请求等待结果即可
	c := &call[V]{}
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()
	//执行请求，执行完毕其余等待请求直接返回结果
	c.val, c.err = fn()
	c.wg.Done()
	//请求结束，删除请求
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
