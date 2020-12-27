package gocache

import pb "gocache/gocachepb"
// 被用于去定位拥有key的节点
type PeerPicker interface {
	PickPeer(key string)(peer PeerGetter,ok bool)
}

// 节点用于获取特定key的接口 对应于之前实现的http客户端
type PeerGetter interface {
	//Get(group string,key string)([]byte,error)
	Get(in *pb.Request,out *pb.Response)error
}