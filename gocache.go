package gocache

import (
	"fmt"
	pb "gocache/gocachepb"
	"gocache/singleflight"
	"log"
	"sync"
)

//                             是
// 接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
//                 |  否                         是
//                 |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
//                             |  否
//                             |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶


type Getter interface{
	Get(key string)([]byte,error) // 回调函数Get
}

// 实现 Getter 接口的 Get 方法
type GetterFunc func(key string)([]byte,error)

//  
func (f GetterFunc)Get(key string)([]byte,error){
	return f(key)
}

// A Group is a cache namespace and associated data loaded spread over
type Group struct {
	name string 
	getter Getter  
	mainCache cache
	peers PeerPicker
	// use singleflight.Group to make sure that
	// each key is only fetched once 保证每个key只被取一次 其他使用第一次的结果
	loader *singleflight.Group

}

var (
	mu sync.RWMutex
	groups=make(map[string]*Group)
)

func NewGroup(name string,cacheBytes int64,getter Getter)*Group{
	if getter==nil{
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock() 
	g:=&Group{
		name:name,
		getter: getter,
		mainCache: cache{cacheBytes:cacheBytes},
		loader: &singleflight.Group{},
	}
	groups[name]=g
	return g
}

func GetGroup(name string)*Group{
	mu.RLock()
	g:=groups[name]
	mu.RUnlock()
	return g
}
// 注册一个PeerPicker 用于选择远处的节点
func (g *Group)RegisterPeers(peers PeerPicker){
	if g.peers!=nil{
		panic("RegisterPeerPicker called more than once")
	}
	g.peers=peers
}


func (g *Group)Get(key string)(ByteView,error){
	if key==""{
		return ByteView{}, fmt.Errorf("key is required")
	}
	if v,ok:=g.mainCache.get(key);ok{
		//这里是从本地的缓存获取
		log.Println("[GeeCache] hit")
		return v,nil
	}
	// 如果本地的缓存没有 则从load 导入
	return g.load(key)
}

func (g *Group)load(key string)(value ByteView,err error){
	// each key is only fetched once (either locally or remotely)
	// regardless of the number of concurrent callers.
	viewi,err:=g.loader.Do(key, func() (interface{}, error) {
		if g.peers!=nil{
			if peer,ok:=g.peers.PickPeer(key);ok{
				if value,err:=g.getFromPeer(peer,key);err==nil{
					return value,nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}
		// 如果没有远程节点 则从本地获取
		return g.getLocally(key)
	})

	if err==nil{
		return viewi.(ByteView),nil
	}
	return
}

func (g *Group)getFromPeer(peer PeerGetter,key string)(ByteView,error){
	req:=&pb.Request{Group: g.name,Key: key}
	res:=&pb.Response{}

	err:=peer.Get(req,res)
	if err!=nil{
		return ByteView{},err
	}
	return ByteView{b: res.Value},nil
}

func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err

	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

// 填充缓存 将key:value 填充缓存
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}

