package lru

import "sync"

type lruNode struct {
	key  []byte
	prev *lruNode
	next *lruNode
}
type LRU struct {
	capacity int
	size     int

	head *lruNode // 最近使用
	tail *lruNode // 最久未使用

	table map[string]*lruNode // key(string) -> node
	mu    sync.Mutex
}

func NewLRU(capacity int) *LRU {
	if capacity <= 0 {
		panic("lru capacity must > 0")
	}
	return &LRU{
		capacity: capacity,
		table:    make(map[string]*lruNode),
	}
}
func (l *LRU) moveToHead(node *lruNode) {
	if node == l.head {
		return
	}

	l.removeNode(node)
	l.addToHead(node)
}

func (l *LRU) addToHead(node *lruNode) {
	node.prev = nil
	node.next = l.head

	if l.head != nil {
		l.head.prev = node
	}
	l.head = node

	if l.tail == nil {
		l.tail = node
	}
}

func (l *LRU) removeNode(node *lruNode) {
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		l.head = node.next
	}

	if node.next != nil {
		node.next.prev = node.prev
	} else {
		l.tail = node.prev
	}

	node.prev = nil
	node.next = nil
}
func (l *LRU) Put(key []byte) (evictedKey []byte, evicted bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	k := string(key)

	// 已存在：移动到头部
	if node, ok := l.table[k]; ok {
		l.moveToHead(node)
		return nil, false
	}

	// 新节点
	node := &lruNode{
		key: append([]byte(nil), key...), // 防止外部修改
	}
	l.table[k] = node
	l.addToHead(node)
	l.size++

	// 超出容量，淘汰 tail
	if l.size > l.capacity {
		evictedNode := l.tail
		l.removeNode(evictedNode)
		delete(l.table, string(evictedNode.key))
		l.size--
		return evictedNode.key, true
	}

	return nil, false
}
func (l *LRU) Get(key []byte) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if node, ok := l.table[string(key)]; ok {
		l.moveToHead(node)
		return true
	}
	return false
}
func (l *LRU) Delete(key []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()

	k := string(key)
	if node, ok := l.table[k]; ok {
		l.removeNode(node)
		delete(l.table, k)
		l.size--
	}
}
func (l *LRU) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.head = nil
	l.tail = nil
	l.size = 0
	l.table = make(map[string]*lruNode)
}
