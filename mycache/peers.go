package mycache

import "mycache/cachepb"

/*
节点选择器，通过key选择应访问的节点
*/
type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter,ok bool)
}

/*
节点的数据获取接口，通过pb通讯
*/
type PeerGetter interface {
	Get(in *cachepb.Request, out *cachepb.Response) error
}


