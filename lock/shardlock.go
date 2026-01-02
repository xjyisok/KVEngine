// key_shard_lock.go
package bitcaskgo

import "sync"

type ShardedLock struct {
	locks []sync.RWMutex
	shard int
}

func NewShardedLock(n int) *ShardedLock {
	return &ShardedLock{
		locks: make([]sync.RWMutex, n),
		shard: n,
	}
}

func (s *ShardedLock) hash(key []byte) int {
	h := uint32(2166136261)
	for _, b := range key {
		h = (h ^ uint32(b)) * 16777619
	}
	return int(h % uint32(s.shard))
}

func (s *ShardedLock) Lock(key []byte) {
	s.locks[s.hash(key)].Lock()
}

func (s *ShardedLock) Unlock(key []byte) {
	s.locks[s.hash(key)].Unlock()
}

func (s *ShardedLock) RLock(key []byte) {
	s.locks[s.hash(key)].RLock()
}

func (s *ShardedLock) RUnlock(key []byte) {
	s.locks[s.hash(key)].RUnlock()
}
