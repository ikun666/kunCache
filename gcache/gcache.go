package gcache

import (
	"kunCache/cache"
	"kunCache/conf"
	"kunCache/peer"
	"log/slog"
	"sync"
	"time"

	"kunCache/singleflight"
)

// A Getter loads data for a key.
type Getter[K comparable, V any] interface {
	Get(key K) (V, error)
}

// A GetterFunc implements Getter with a function.
type GetterFunc[K comparable, V any] func(key K) (V, error)

// 接口型函数 函数绑定方法实现接口调用自己
func (f GetterFunc[K, V]) Get(key K) (V, error) {
	return f(key)
}

// 分组cache
type Group[K comparable, V any] struct {
	name      string
	getter    Getter[K, V]
	mainCache *cache.Cache[K, V]
	//分布式节点
	peers peer.Picker[K, V]
	//并发请求同一个key只执行一次
	loader *singleflight.Group[K, V]
}

var (
	mu     sync.RWMutex
	groups = make(map[string]any)
)

// NewGroup create a new instance of Group
func NewGroup[K comparable, V any](name string, maxEntries int64, getter Getter[K, V]) *Group[K, V] {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group[K, V]{
		name:      name,
		getter:    getter,
		mainCache: cache.New[K, V](maxEntries, nil),
		loader:    &singleflight.Group[K, V]{},
	}
	groups[name] = g
	return g
}

// GetGroup returns the named group previously created with NewGroup, or
// nil if there's no such group.
func GetGroup[K comparable, V any](name string) *Group[K, V] {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g.(*Group[K, V])
}

// RegisterServer registers a PeerPicker for choosing remote peer
func (g *Group[K, V]) RegisterServer(peers peer.Picker[K, V]) {
	if g.peers != nil {
		slog.Error("[RegisterPeerPicker called more than once]")
		return
	}
	g.peers = peers
}

// Get value for a key from cache
// 没有缓存会调用回调函数加载
func (g *Group[K, V]) Get(key K) (V, error) {
	if v, ok := g.mainCache.Get(key); ok {
		slog.Info("[GCache] hit")
		// fmt.Println("cache", v)
		return v, nil
	}

	return g.load(key)
}

// 没有缓存  可选本地和远端加载
func (g *Group[K, V]) load(key K) (V, error) {
	value, err := g.loader.Do(key, func() (V, error) {
		//优先从远端加载缓存
		if g.peers != nil {
			if p, ok := g.peers.Pick(key); ok {
				value, err := g.getFromPeer(p, key)
				if err == nil {
					return value, nil
				}
				slog.Info("[GCache] Failed to get from peer", "err", err)
			}
			slog.Info("[GCache] Failed to get from peer")
			return g.getLocally(key)
		}
		return g.getLocally(key)
	})
	if err != nil {
		return value, err
	}
	return value, err
}

// 从远端加载数据
func (g *Group[K, V]) getFromPeer(peer peer.Fetcher[K, V], key K) (V, error) {
	value, err := peer.Fetch(g.name, key)
	if err != nil {
		return value, err
	}
	return value, nil
}

// 从本地加载数据
func (g *Group[K, V]) getLocally(key K) (V, error) {
	value, err := g.getter.Get(key)
	if err != nil {
		return value, err

	}
	// fmt.Println("local", value)
	g.populateCache(key, value, time.Now().Add(time.Duration(conf.GConfig.Expires)*time.Minute).UnixNano())
	return value, nil
}

// 加载到缓存
func (g *Group[K, V]) populateCache(key K, value V, expires int64) {
	g.mainCache.Add(key, value, expires)
}
