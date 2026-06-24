package core

import (
	"github.com/redis-server/alloc"
	"github.com/redis-server/config"
)

var store map[string]Obj
var expires map[Obj]uint64
var ePool *EvictionPool
var registry map[string]Command

func Init() {
	store = make(map[string]Obj)
	expires = make(map[Obj]uint64)
	ePool = newEvictionPool(16)

	registry = map[string]Command{
		"SET":          SetCmd{},
		"GET":          GetCmd{},
		"TTL":          TtlCmd{},
		"DEL":          DelCmd{},
		"EXPIRE":       ExpCmd{},
		"BGREWRITEAOF": AOFCmd{},
		"INCR":         IncrCmd{},
		"INFO":         InfoCmd{},
		"CLIENT":       ClientCmd{},
		"LATENCY":      LatencyCmd{},
		"SLEEP":        SleepCmd{},
		"RPUSH":        RPUSHCmd{},
		"LPUSH":        LPUSHCmd{},
		"LLEN":         LLENCmd{},
		"LRANGE":       LRANGECmd{},
		"LPOP":         LPOPCmd{},
		"BF.ADD":       BFADDCmd{},
		"BF.EXISTS":    BFEXISTSCmd{},
		"BF.RESERVE":   BFRESERVECmd{},
		"SADD":         SaddCmd{},
		"SCARD":        ScardCmd{},
		"SISMEMBER":    SismemberCmd{},
		"SMEMBERS":     SmembersCmd{},
		"SREM":         SremCmd{},
		"ZADD":         ZaddCmd{},
		"ZREM":         ZremCmd{},
		"ZSCORE":       ZscoreCmd{},
		"ZRANGE":       ZrangeCmd{},
		"SUBSCRIBE":    SubCmd{},
		"UNSUBSCRIBE":  UnsubCmd{},
		"PUBLISH":      PublishCmd{},
		"BLPOP":        BlpopCmd{},
		"WATCH":        WatchCmd{},
		"MULTI":        MultiCmd{},
		"EXEC":         ExecCmd{},
		"DISCARD":      DiscardCmd{},
		"PING":         PingCmd{},
		"GEOADD":       GeoAddCmd{},
		"GEOHASH":      GeoHashCmd{},
		"GEODIST":      GeoDistCmd{},
		"GEOPOS":       GeoPosCmd{},
		"GEOSEARCH":    GeoSearchCmd{},
	}
}

func Lookup(name string) (Command, bool) {
	cmd, ok := registry[name]
	return cmd, ok
}

func NewObj(value interface{}, expDurationMs int64, oType uint8, oEnc uint8) Obj {

	obj := Obj{
		Value:          value,
		TypeEncoding:   oType | oEnc,
		LastAccessedAt: getCurrentClock(),
	}

	if expDurationMs > 0 {
		setExpiry(obj, expDurationMs)
	}

	return obj
}

func Put(k string, obj Obj) {

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

	SignalModifiedKey(k)

	KeyspaceStat[0]["keys"]++
}

func Get(k string) (Obj, bool) {
	v, ok := store[k]
	if !ok {
		return v, ok
	}
	if ok {
		if hasExpired(v) {
			Del(k)
			return v, false
		}
	}
	v.LastAccessedAt = getCurrentClock()
	return v, true
}

func Del(k string) bool {

	if obj, ok := store[k]; ok {
		delete(store, k)
		delete(expires, obj)
		KeyspaceStat[0]["keys"]--

		mem := obj.Size() + int64(len(k))
		alloc.Free(make([]byte, 0, mem))

		SignalModifiedKey(k)

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

func IsEmpty(key string) bool {

	obj, exists := store[key]

	if !exists {
		return true
	}

	ql := obj.Value.(*Quicklist)

	return ql.len == 0
}
