package core

import (
	"github.com/redis-server/alloc"
	"github.com/redis-server/config"
)

var store map[string]*Obj
var expires map[*Obj]uint64
var ePool *EvictionPool

func Init() {
	store = make(map[string]*Obj)
	expires = make(map[*Obj]uint64)
	ePool = newEvictionPool(16)
}

func NewObj(value interface{}, expDurationMs int64, oType uint8, oEnc uint8) *Obj {

	obj := Obj{
		Value:          value,
		TypeEncoding:   oType | oEnc,
		LastAccessedAt: getCurrentClock(),
	}

	if expDurationMs > 0 {
		setExpiry(&obj, expDurationMs)
	}

	return &obj
}

func Put(k string, obj *Obj) {
	if len(store) >= config.KeyLimit {
		evict()
	}

	mem := obj.Size() + int64(len(k))

	_, err := alloc.Alloc(int(mem))
	if err != nil {
		return
	}

	obj.LastAccessedAt = getCurrentClock()
	store[k] = obj

	if KeyspaceStat[0] == nil {
		KeyspaceStat[0] = make(map[string]int)
	}
	KeyspaceStat[0]["keys"]++
}

func Get(k string) *Obj {
	v := store[k]
	if v == nil {
		return nil
	}
	if v != nil {
		if hasExpired(v) {
			Del(k)
			return nil
		}
	}
	v.LastAccessedAt = getCurrentClock()
	return v
}

func Del(k string) bool {

	if obj, ok := store[k]; ok {
		delete(store, k)
		delete(expires, obj)
		KeyspaceStat[0]["keys"]--

		mem := obj.Size() + int64(len(k))
		alloc.Free(make([]byte, 0, mem))

		return true
	}
	return false
}

func DeleteExpiredKey() {
	for {
		frac := expireSample()

		if frac < 0.25 {
			break
		}
	}

	// log.Println("deleted the expired but undeleted keys. total keys", len(store))
}
