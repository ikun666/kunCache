package peer

// Picker 定义了获取分布式节点的能力
type Picker[K comparable, V any] interface {
	Pick(key K) (Fetcher[K, V], bool)
	AddPeers(peersAddr ...K)
	DelPeers(peersAddr ...K)
}

// Fetcher 定义了从远端获取缓存的能力，所以每个 Peer 都应实现这个接口
type Fetcher[K comparable, V any] interface {
	Fetch(group string, key K) (V, error)
}
