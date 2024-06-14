package consistentHash

import (
	"fmt"
	"hash/crc32"
	"slices"
)

// Hash maps bytes to uint32
// hash函数
type Hash func(data []byte) uint32

// Map constains all hashed keys
type Map[K comparable] struct {
	//hash函数
	hash Hash
	//虚拟节点倍数
	replicas int
	//有序哈希环
	hashRing []uint32
	//虚拟节点与真实节点的映射表
	hashMap map[uint32]K
}

// New creates a Map instance
func New[K comparable](replicas int, hash Hash) *Map[K] {
	m := &Map[K]{
		replicas: replicas,
		hash:     hash,
		hashMap:  make(map[uint32]K),
	}
	//不指定哈希函数使用默认哈希
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add adds some keys to the hash.
// 添加keys到哈希环
func (m *Map[K]) Add(keys ...K) {
	for _, key := range keys {
		//每个key都要有虚拟节点 加入虚拟节点
		for i := 0; i < m.replicas; i++ {
			hash := m.hash([]byte(fmt.Sprintf("%v%v", i, key)))
			m.hashRing = append(m.hashRing, hash)
			m.hashMap[hash] = key
		}
	}
	slices.Sort(m.hashRing)
	// fmt.Println(m.hashRing)
}

// Remove use to remove a key and its virtual keys on the ring and map
func (m *Map[K]) Remove(keys ...K) {
	for _, key := range keys {
		//删除虚拟节点
		for i := 0; i < m.replicas; i++ {
			hash := m.hash([]byte(fmt.Sprintf("%v%v", i, key)))
			idx, ok := slices.BinarySearch(m.hashRing, hash)
			if !ok {
				continue
			}
			m.hashRing = append(m.hashRing[:idx], m.hashRing[idx+1:]...)
			delete(m.hashMap, hash)
		}
	}
	// fmt.Println(m.hashRing)

}

// Get gets the closest item in the hash to the provided key.
func (m *Map[K]) Get(key K) (addr K) {
	if len(m.hashRing) == 0 {
		return
	}

	hash := m.hash([]byte(fmt.Sprintf("%v", key)))
	//二分查找找第一个大于等于目标值的
	left, right := 0, len(m.hashRing)-1
	for left <= right {
		mid := left + (right-left)>>1
		if m.hashRing[mid] >= hash {
			right = mid - 1
		} else {
			left = mid + 1
		}
	}
	//没有找到大于等于的  取顺时针第一个 映射成真实节点
	if left == len(m.hashRing) {
		return m.hashMap[m.hashRing[0]]
	}

	return m.hashMap[m.hashRing[left]]

}
