package lru

import (
	"container/list"
)

type Cache struct {
	maxBytes  int64                  //最大长度
	lenBytes  int64                  //当前长度
	nodeList  *list.List
	cache     map[string]*list.Element
	onEvicted func(key string,value Value)
}

type Entry struct {
	key   string
	value Value
}

type Value interface {
	Len() int
}

func New(max int64,onevicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  max,
		nodeList:  list.New(),
		cache:     make(map[string]*list.Element),
		onEvicted: onevicted,
	}
}

func (this *Cache)Get(key string) (value Value,ok bool) {
	if node,ok := this.cache[key];ok {
		this.nodeList.MoveToFront(node)
		entry := node.Value.(*Entry)
		return entry.value,true
	}
	return
}

func (this *Cache)RemoveOldest() {
	node := this.nodeList.Back()
	if node != nil {
		this.nodeList.Remove(node)
		entry := node.Value.(*Entry)
		delete(this.cache,entry.key)
		this.lenBytes -= int64(len(entry.key)) + int64(entry.value.Len())
		if this.onEvicted != nil {
			this.onEvicted(entry.key,entry.value)
		}
	}
}

func (this *Cache)Add(key string,value Value) {
	//if exist in map,update the node
	if node,ok := this.cache[key];ok {
		this.nodeList.MoveToFront(node)
		entry := node.Value.(*Entry)
		this.lenBytes += int64(value.Len()) - int64(entry.value.Len())
		entry.value = value
	}else{
	//not exist in map,create node
		node := this.nodeList.PushFront(&Entry{key: key,value: value})
		this.cache[key] = node
		this.lenBytes += int64(len(key)) + int64(value.Len())
	}
	//if out of max length,remove oldest node
	for this.maxBytes != 0 && this.lenBytes > this.maxBytes {
		this.RemoveOldest()
	}
}

func (this *Cache)Len() int {
	return this.nodeList.Len()
}

