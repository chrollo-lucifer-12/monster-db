package core

import "time"

func getCurrentClock() uint32 {
	return uint32(time.Now().Unix()) & 0x00FFFFFF
}

func setExpiry(key string, expDurationMs int64) {
	expires[key] = uint64(time.Now().UnixMilli()) + uint64(expDurationMs)
}

func hasExpired(key string) bool {
	exp, ok := expires[key]

	if !ok {
		return false
	}

	return exp <= uint64(time.Now().UnixMilli())
}

func getExpiry(key string) (uint64, bool) {
	exp, ok := expires[key]
	return exp, ok
}

func expireSample() float32 {
	var limit int = 20
	var expiredCount int = 0

	for key, _ := range store {
		if hasExpired(key) {
			limit--
			Del(key)
		}

		if limit == 0 {
			break
		}
	}

	return float32(expiredCount) / float32(20.0)
}
