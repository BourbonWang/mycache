# mycache
参考groupcache，实现分布式LRU缓存。通过Protobuf通信。
##实现
* 通过双链表和map实现底层LRU
* 加入锁以达到并发安全
* 一致性哈希选择节点
* 支持同时运行多个缓存实例
* 实现单点反射，防止缓存被击穿
* 使用protobuf通信

