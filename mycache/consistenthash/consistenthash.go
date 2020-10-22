package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32

type Map struct {
	hash     Hash            //哈希函数
	replicas int             //虚拟节点倍数
	keys     []int			 //哈希环
	hashMap  map[int]string  //哈希值-节点映射
}

func New(replicas int,hashFunc Hash) *Map {
	m := &Map{
		hash:     hashFunc,
		replicas: replicas,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

func (this *Map) Add(keys ...string) {
	for _,key := range keys {
		for i := 0;i < this.replicas;i++ {
			hash := int(this.hash([]byte(strconv.Itoa(i)+key)))
			this.keys = append(this.keys,hash)
			this.hashMap[hash] = key
		}
	}
	sort.Ints(this.keys)
}

func (this *Map) Get(key string) string {
	if len(this.keys) == 0 {
		return ""
	}
	hash := int(this.hash([]byte(key)))
	idx := sort.Search(len(this.keys), func(i int) bool {
		return this.keys[i] >= hash
	})
	return this.hashMap[this.keys[idx%(len(this.keys))]]
}
