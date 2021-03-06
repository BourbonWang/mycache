# mycache
分布式LRU缓存，通过key-value存储。支持并发访问，通过Protobuf通信。  
+ 通过key选择该访问的节点
+ 缓存获取不到时调用用户接口进行缓存填充，通过LRU机制进行缓存淘汰
+ 每个key对应唯一节点，数据仅在一台节点上缓存，保证集群不会被热点数据占满
+ 支持大量并发访问。当大规模请求数据，锁机制保证仅对缓存请求一次，减轻节点压力
+ 不支持缓存更新，不支持定时淘汰
+ 每台节点运行代码完全相同，方便部署
+ 不需要额外的客户端，客户端本身也是缓存服务器，与自身进行通信
+ 可以分布式部署，也可以单机多端口部署  
## 数据获取流程  
用户请求获取键key，mycache将：
1. 客户端向本机缓存节点请求数据
2. 检查对于同一个key，是否已有请求正在处理。若有，等待已有请求的结果直接返回; 否则：
3. 一致性哈希通过key选择是否从远端节点获取value，若是，与对应节点通信并返回value; 否则：
4. 查找本机缓存
5. 若缓存未命中，从外部数据接口获取，并更新缓存
## 实现  
本缓存学习[groupcache](https://github.com/golang/groupcache), golang实现。  
1. 实现LRU，加入锁以达到并发安全
3. 一致性哈希选择节点，避免缓存雪崩
4. 实现同时运行多个缓存实例，封装缓存填充接口
5. 实现单点反射，防止缓存被击穿
6. http服务器，使用protobuf通信
## 使用  
### 创建缓存group
```go
  group := mycache.NewGroup(groupName, maxSize, mycache.GetterFunc(
		//缓存未命中时的数据获取函数,数据库等
		func(key string) ([]byte, error) {
        	//search from DB
			return []byte(value), err
		}))
```
### 启动缓存服务器
addr:  本机地址 如：aa.bb.c.ddd:port  
addrs: 所有节点地址列表
```go
  peers := mycache.NewHTTPPool(addr)
  peers.Set(addrs...)
  group.RegisterPeers(peers)
  http.ListenAndServe("0.0.0.0:"+port, peers)  
```
### 启动客户端
部署时，在其中某个节点调用
```go
  http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := group.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())
		}))    
    http.ListenAndServe(addr + clientPort, nil)
```
用户访问 ip:clientPort/api?key=xxx，客户端直接调用本地节点服务器。

## 部署与测试  
以在本机多端口部署为例，集群部署同理。
如：缓存节点部署在8001，8002,8003端口，客户端在9999端口。具体见[main.go](https://github.com/BourbonWang/mycache/blob/master/main.go)
```go
func main() {
	var port int  //缓存服务的端口号
	var api bool  //是否将本设备作为用户访问节点
	flag.IntVar(&port, "port", 8001, "myCache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
	flag.Parse()

	//用于用户访问的对外服务端口
	apiAddr := "http://0.0.0.0:9999"
	//缓存服务的所有节点
	addrMap := map[int]string{
		//你的本机ip
		8001: "http://x.x.x.x:8001",
		8002: "http://x.x.x.x:8002",
		8003: "http://x.x.x.x:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	group := createGroup()
	if api {		//如果在此设备上开启用户外部访问服务
		go startAPIServer(apiAddr, group)
	}
	startCacheServer(addrMap[port], addrs, group)
}
```
### docker部署
```
docker run --name  node1 -p 8001:8001 mycache /server -port=8001
docker run --name  node1 -p 8002:8002 mycache /server -port=8002
docker run --name  node1 -p 8003:8003 -p 9999:9999 mycache /server -port=8003 -api=1
```
### 测试
```
curl "http://localhost:9999/api?key=xxx" &
```
## 后续完善  
+ 客户端节点故障后，自动选举新节点
+ 实现动态增加、减少节点

