package core

import (
	"github.com/redis-server/config"
)

var EPool *EvictionPool

func getIdleTime(lastAccessedAt uint32) uint32 {
	c := getCurrentClock()
	if c >= lastAccessedAt {
		return c - lastAccessedAt
	}
	return (0x00FFFFFF - lastAccessedAt) + c
}

func populateEvictionPool() {
	sampleSize := 5

	for k := range store {
		EPool.Push(k, store[k].LastAccessedAt)
		sampleSize--
		if sampleSize == 0 {
			break
		}
	}
}

func evictFirst() {
	for k := range store {
		delete(store, k)
		return
	}
}

func evictAllKeysRandom() {
	evictCount := int64(config.EvictionRatio * float64(config.KeyLimit))

	for k := range store {
		Del(k)
		evictCount--
		if evictCount <= 0 {
			break
		}
	}
}

func evictAllkeysLRU() {
	populateEvictionPool()

	evictCount := int16(config.EvictionRatio * float64(config.KeyLimit))

	for i := 0; i < int(evictCount) && len(EPool.pool) > 0; i++ {
		item := EPool.Pop()
		if item == nil {
			return
		}
		Del(item.key)
	}
}

func evict() {

	evictAllkeysLRU()
}
