# mycache
分布式LRU缓存，通过key-value存储。支持并发访问，通过Protobuf通信。  
+ 通过key选择该访问的节点
+ 缓存获取不到时调用用户接口进行缓存填充，通过LRU机制进行缓存淘汰
+ 每个key对应唯一节点，数据仅在一台节点上缓存，保证集群不会被热点数据占满
+ 不支持缓存更新，不支持定时淘汰
+ 每台节点运行代码完全相同，方便部署
+ 不需要额外的客户端，客户端本身也是缓存服务器，与自身进行通信
+ 可以分布式部署，也可以单机多端口部署

## 实现  
本缓存学习groupcache(https://github.com/golang/groupcache), golang实现。  
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

## 后续完善  
+ 客户端节点故障后，自动选举新节点
+ 动态增加、减少节点

