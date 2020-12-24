package lru 

import "container/list" 

//  Cache is a LRU csche,It is not sfae for concurrent access 
type Cache struct{
	maxBytes int64  //允许使用的最大内存
	nbytes int64 
	ll *list.List 
	cache map[string]*list.Element
	// optional and executed when an entry is purged. 某条记录被移除时的回调函数 
	OnEvicted func(key string, value Value)
}

type entry struct {
	key   string
	value Value // 为了通用性 value是任意实现了value接口的类型 
				// value接口 的len方法 返回值所占用的内存大小。
}

// Value use Len to count how many bytes it takes
type Value interface {
	Len() int
}

// New  实例化
func New(maxBytes int64,onEvicted func(string,Value))*Cache{
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// 查找功能 主要有两个步骤 第一步是从字典中找到对应的双向链表的节点，第二步，将该节点移动到队头

func (c *Cache)Get(key string)(value Value,ok bool){
	if ele,ok:=c.cache[key];ok{
		c.ll.MoveToFront(ele)
		kv:=ele.Value.(*entry)
		return kv.value,true
	}
	return
}

// 删除 即淘汰缓存 移除最近最少访问的节点 也就是队尾的节点 

func (c *Cache)RemoveOldest(){
	ele:=c.ll.Back()
	if ele!=nil{
		c.ll.Remove(ele) // 取队尾节点 从链表删除
		kv:=ele.Value.(*entry)
		delete(c.cache,kv.key) //从字典中 c.cache 删除该节点的映射关系。
		c.nbytes-=int64(len(kv.key))+int64(kv.value.Len()) // 更新当前所用的内存 c.nbytes。
		if c.OnEvicted!=nil{ //如果回调函数 OnEvicted 不为 nil，则调用回调函数。
			c.OnEvicted(kv.key,kv.value)
		}
	}
}

func (c *Cache)Add(key string,value Value){
	if ele,ok:=c.cache[key];ok{
		c.ll.MoveToFront(ele)
		kv:=ele.Value.(*entry)
		c.nbytes+=int64(value.Len()) - int64(kv.value.Len())
		kv.value=value
	}else{
		ele:=c.ll.PushFront(&entry{key,value})
		c.cache[key]=ele 
		c.nbytes+=int64(len(key))+int64(value.Len())
	}
	for c.maxBytes!=0&&c.maxBytes<c.nbytes{
		c.RemoveOldest()
	}
}

func (c *Cache)Len()int{
	return c.ll.Len()
}