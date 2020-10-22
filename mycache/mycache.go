package mycache

import (
	"fmt"
	"log"
	"mycache/cachepb"
	"mycache/lru"
	"mycache/singleflight"
	"sync"
)

//缓存未命中时，获取数据的函数接口
type Getter interface {
	Get(key string) ([]byte,error)
}

//实现Getter接口，并调用自身
type GetterFunc func(key string) ([]byte,error)

func (f GetterFunc) Get(key string)([]byte,error){
	return f(key)
}


/*
缓存的主体数据结构
实现从peer获取缓存
*/
type Group struct {
	name      string       //命名空间，以区分多个缓存
	getter    Getter       //未命中时，从外部数据源获取数据的接口
	mainCache cache        //缓存
	peers     PeerPicker   //子节点选择接口
	loader    *singleflight.Group
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func NewGroup(name string,lenbytes int64,getter Getter) *Group {
	if getter == nil {
		panic("nil getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{lenBytes: lenbytes},
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

//通过命名空间，获取group
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

//注册peerPicker，只能调用一次
func (this *Group) RegisterPeers(peers PeerPicker) {
	if this.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	this.peers = peers
}

//获取数据
func (this *Group) Get(key string) (ByteView,error) {
	if key == "" {
		return ByteView{},fmt.Errorf("nil key")
	}
	if v,ok := this.mainCache.get(key);ok {
		log.Println("[myCache] "+key+" : hit")
		return v,nil
	}
	return this.load(key)
}

func (this *Group) load(key string) (ByteView,error) {
	val,err := this.loader.Do(key, func() (interface{}, error) {
		//选择是否从子节点获取
		if this.peers != nil {
			if peer,ok := this.peers.PickPeer(key);ok {
				value,err :=  this.getFromPeer(peer,key)
				if err == nil {
					return value,nil
				}
				log.Println("[myCache] Failed to get from peer",err)
			}
		}
		//if not,get value locally
		return this.getLocally(key)
	})

	if err != nil {
		return ByteView{},err
	}
	return val.(ByteView),nil
}

//从分布式节点获取
func (this *Group) getFromPeer(peer PeerGetter,key string) (ByteView,error) {
	req := &cachepb.Request{
		Group: this.name,
		Key:   key,
	}
	res := &cachepb.Response{}

	err := peer.Get(req,res)
	if err != nil {
		return ByteView{},err
	}
	return ByteView{res.Value},nil
}

//从本地getter获取
func (this *Group) getLocally(key string) (ByteView,error) {
	bytes,err := this.getter.Get(key)
	if err != nil {
		return ByteView{},err
	}
	value := ByteView{b: cloneBytes(bytes)}
	this.populateCache(key,value)
	return value,nil
}

//向缓存中添加数据
func (this *Group) populateCache(key string,value ByteView) {
	this.mainCache.add(key,value)
}


/*
对lru.Cache的封装
加入并发锁，用ByteView封装数据类型
*/
type cache struct {
	mu       sync.Mutex
	lru      *lru.Cache
	lenBytes int64
}

func (this *cache) add(key string,value ByteView) {
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.lru == nil {
		this.lru = lru.New(this.lenBytes,nil)
	}
	this.lru.Add(key,value)
}

func (this *cache) get(key string) (value ByteView,ok bool) {
	this.mu.Lock()
	defer this.mu.Unlock()
	if this.lru == nil {
		return
	}
	if v,ok := this.lru.Get(key);ok {
		return v.(ByteView),true
	}
	return
}
