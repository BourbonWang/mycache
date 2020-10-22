package singleflight

import (
	"sync"
)

/*
请求的结构体，包含等待队列，返回数据，返回错误
*/
type call struct {
	wg  sync.WaitGroup  //等待
	val interface{}
	err error
}

/*
对所有请求建立缓存，通过key标记
*/
type Group struct {
	mu sync.Mutex
	m  map[string]*call
}

func (this *Group) Do(key string,fn func() (interface{},error)) (interface{},error) {
	this.mu.Lock()
	if this.m == nil {
		this.m = make(map[string]*call)
	}
	//如果请求已经在队列中，等待其结果，直接返回
	if c,ok := this.m[key];ok {
		this.mu.Unlock()
		c.wg.Wait()        //等待结果
		return c.val,c.err
	}
	//未请求过，新建
	c := new(call)
	c.wg.Add(1)
	this.m[key] = c
	this.mu.Unlock()

	c.val,c.err = fn()   //执行函数
	c.wg.Done()          //释放

	this.mu.Lock()
	delete(this.m,key)
	this.mu.Unlock()

	return c.val,c.err
}