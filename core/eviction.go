package core

import "github.com/redis-server/config"

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

func evict() {
	switch config.EvictionStrategy {
	case "allkeys-random":
		evictAllKeysRandom()
	default:
		evictFirst()
	}
}
