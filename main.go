package main

import (
	"flag"
	"fmt"
	"log"
	"mycache"
	"net/http"
	"time"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

//创建缓存group
func createGroup() *mycache.Group {
	return mycache.NewGroup("scores", 2<<10, mycache.GetterFunc(
		//缓存未命中时的数据获取函数
		func(key string) ([]byte, error) {
			time.Sleep(1 * time.Second)
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

//启动缓存服务器，实例化HTTPPool
func startCacheServer(addr string, addrs []string, group *mycache.Group) {
	peers := mycache.NewHTTPPool(addr)
	//将节点加入一致性哈希
	peers.Set(addrs...)
	//注册节点选择器 PeerPicker
	group.RegisterPeers(peers)
	log.Println("myCache is running at", addr)

	log.Fatal(http.ListenAndServe("0.0.0.0:"+addr[22:], peers))
}

//外部接口：用于用户交互，访问 http://ip:port/api?key=xx
func startAPIServer(apiAddr string, group *mycache.Group) {
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
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))

}

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
		8001: "http://10.234.113.126:8001",
		8002: "http://10.234.113.126:8002",
		8003: "http://10.234.113.126:8003",
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