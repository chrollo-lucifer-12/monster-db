package core

import (
	"log"
	"time"

	"github.com/redis-server/config"
)

var store map[string]*Obj

func Init() {
	store = make(map[string]*Obj)
}

func NewObj(value interface{}, durationMs int64, oType uint8, oEnc uint8) *Obj {
	var expiresAt int64 = -1
	if durationMs > 0 {
		expiresAt = time.Now().UnixMilli() + durationMs
	}

	return &Obj{
		Value:        value,
		TypeEncoding: oType | oEnc,
		ExpiresAt:    expiresAt,
	}
}

func Put(k string, obj *Obj) {
	if len(store) >= config.KeyLimit {
		evict()
	}
	store[k] = obj
}

func Get(k string) *Obj {
	v := store[k]
	if v != nil {
		if v.ExpiresAt != -1 && v.ExpiresAt <= time.Now().UnixMilli() {
			delete(store, k)
			return nil
		}
	}
	return v
}

func Del(k string) bool {
	if _, ok := store[k]; ok {
		delete(store, k)
		return true
	}
	return false
}

func expireSample() float32 {
	var limit int = 20
	var expiredCount int = 0

	for key, obj := range store {
		if obj.ExpiresAt != -1 {
			limit--

			if obj.ExpiresAt <= time.Now().UnixMilli() {
				delete(store, key)
				expiredCount++
			}
		}

		if limit == 0 {
			break
		}
	}

	return float32(expiredCount) / float32(20.0)
}

func DeleteExpiredKey() {
	for {
		frac := expireSample()

		if frac < 0.25 {
			break
		}
	}

	log.Println("deleted the expired but undeleted keys. total keys", len(store))
}
