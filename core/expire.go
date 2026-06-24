package core

import "time"

func getCurrentClock() uint32 {
	return uint32(time.Now().Unix()) & 0x00FFFFFF
}

func setExpiry(obj Obj, expDurationMs int64) {
	expires[obj] = uint64(time.Now().UnixMilli()) + uint64(expDurationMs)
}

func hasExpired(obj Obj) bool {
	exp, ok := expires[obj]

	if !ok {
		return false
	}

	return exp <= uint64(time.Now().UnixMilli())
}

func getExpiry(obj Obj) (uint64, bool) {
	exp, ok := expires[obj]
	return exp, ok
}

func expireSample() float32 {
	var limit int = 20
	var expiredCount int = 0

	for key, obj := range store {
		if hasExpired(obj) {
			limit--
			Del(key)
		}

		if limit == 0 {
			break
		}
	}

	return float32(expiredCount) / float32(20.0)
}
