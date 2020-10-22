package mycache

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"log"
	"mycache/cachepb"
	"mycache/consistenthash"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

const defaultBasePath = "/cache/"  //默认基地址 http://localhost:xxxx/cache/
const defaultReplicas = 50         //默认虚拟节点倍数


type HTTPPool struct {
	self 	 	string                 //本地地址
	basePath 	string
	mu       	sync.Mutex
	peers    	*consistenthash.Map    //一致性哈希，存储子节点
	httpGetters map[string]*httpGetter //子节点->peerGetter
}

func NewHTTPPool(addr string) *HTTPPool {
	return &HTTPPool{
		self:     addr,
		basePath: defaultBasePath,
	}
}

//创建子节点hashmap
func (this *HTTPPool) Set(peers ...string) {
	this.mu.Lock()
	defer this.mu.Unlock()
	//创建一致性哈希
	this.peers = consistenthash.New(defaultReplicas,nil)
	this.peers.Add(peers...)
	//创建map：子节点ip->httpGetter
	this.httpGetters = make(map[string]*httpGetter)
	for _,peer := range peers {
		this.httpGetters[peer] = &httpGetter{baseURL: peer + this.basePath}
	}
}

//实现PeerPicker接口，通过key获取peerGetter函数
func (this *HTTPPool) PickPeer(key string) (PeerGetter,bool) {
	this.mu.Lock()
	defer this.mu.Unlock()
	//通过哈希得到子节点ip，返回相应的httpGetter
	if peer := this.peers.Get(key);peer != "" && peer != this.self {
		log.Printf("Pick peer %s",peer)
		return this.httpGetters[peer],true
	}
	return nil,false
}

func (this *HTTPPool) ServeHTTP(w http.ResponseWriter,r *http.Request){
	//基地址不匹配
	if !strings.HasPrefix(r.URL.Path,this.basePath) {
		panic("HTTPPool serving unexpected path: "+r.URL.Path)
	}
	log.Printf("[server %s] %s: %s",this.self,r.Method,r.URL.Path)

	//  获取/group/key
	parts := strings.SplitN(r.URL.Path[len(this.basePath):],"/",2)
	if len(parts) != 2 {
		http.Error(w,"bad request",http.StatusBadRequest)
		return
	}
	groupName := parts[0]
	key := parts[1]

	group := GetGroup(groupName)
	if group == nil {
		http.Error(w,"no such group:"+groupName,http.StatusNotFound)
		return
	}

	value,err := group.Get(key)
	if err != nil {
		http.Error(w,err.Error(),http.StatusInternalServerError)
		return
	}

	body,err := proto.Marshal(&cachepb.Response{Value: value.ByteSlice()})
	if err != nil {
		http.Error(w,err.Error(),http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type","application/octet-stream")
	w.Write(body)
}


/*
每个分布式节点ip的结构体，用于获取缓存值
*/
type httpGetter struct {
	baseURL string
}

//实现peerGetter接口,从远程节点获取缓存值
func (this *httpGetter) Get(in *cachepb.Request, out *cachepb.Response) (error) {
	//构造url, baseURL/group/key
	u := fmt.Sprintf(
		"%v%v/%v",
		this.baseURL,
		url.QueryEscape(in.GetGroup()),
		url.QueryEscape(in.GetKey()),
		)
	res,err := http.Get(u)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned: %v",res.Status)
	}

	bytes,err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading response body:%v",err)
	}

	if err = proto.Unmarshal(bytes,out);err != nil {
		return fmt.Errorf("decoding response body: %v",err)
	}
	return nil
}