package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// 用于实现一致性哈希算法

// 将bytes映射为uint32
type Hash func(data []byte)uint32 


// 用于存入所有的哈希过的keys 
type Map struct{
	hash Hash  // 哈希函数 由用户自行定义
	replicas int  //虚拟节点倍数
	keys     []int // Sorted  存的是虚拟的节点 
	hashMap  map[int]string
}

// New creates a Map instance
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// 将节点加入 每个节点会变为多个虚拟节点加入到map中
func (m *Map)Add(keys ...string){
	for _,key:=range keys{
		for i:=0;i<m.replicas;i++{
			hash := int(m.hash([]byte(strconv.Itoa(i) + key))) //通过添加编号的方式区分不同虚拟节点。
			m.keys=append(m.keys, hash)
			m.hashMap[hash]=key
		}
	}
	sort.Ints(m.keys)
}

// 返回key对应的真实节点 先找到虚拟节点的索引 然后虚拟节点和真实节点的映射关系 返回真实节点
func (m *Map)Get(key string)string{
	if len(m.keys)==0{
		return ""
	}
	hash:=int(m.hash([]byte(key)))
	// Binary search for appropriate replica.
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	return m.hashMap[m.keys[idx%len(m.keys)]]

}