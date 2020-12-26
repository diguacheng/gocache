package singleflight

import "sync"


// call 代表正在进行或者已经结束的请求。
type call struct {
	wg sync.WaitGroup // 避免锁重入
	val interface{}
	err error
}

type Group struct {
	mu sync.Mutex // 保护m不被并发地读或者写
	m map[string]*call
}

// 无论Do被调用多少次 fn只被调用一次
func (g *Group)Do(key string,fn func()(interface{},error))(interface{},error){
	g.mu.Lock()
	if g.m==nil{
		g.m=make(map[string]*call)
	}
	if c,ok:=g.m[key];ok{ //如果请求正在进行，则等待
		g.mu.Unlock()
		c.wg.Wait()
		return c.val,c.err // 请求结束 返回结果
	}
	c:=new(call)
	c.wg.Add(1) // wg 用于防止锁的重入
	g.m[key]=c
	g.mu.Unlock()
	c.val,c.err=fn() //调用fn 发起请求
	c.wg.Done()

	g.mu.Lock()
	delete(g.m,key)  // 请求结束后 删除在map里删除这个call
	g.mu.Unlock()
	return c.val,c.err  // 返回结果
}




