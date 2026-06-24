package core

import (
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/redis-server/config"
)

func BenchmarkPut(b *testing.B) {
	store = make(map[string]Obj, b.N)
	config.KeyLimit = math.MaxInt

	obj := Obj{
		TypeEncoding: OBJ_TYPE_STRING | OBJ_ENCODING_RAW,
		Value:        "hello world",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Put(fmt.Sprintf("key-%d", i), obj, -1)
	}
}

func BenchmarkGetHit(b *testing.B) {
	store = make(map[string]Obj, b.N)

	obj := Obj{
		TypeEncoding: OBJ_TYPE_STRING | OBJ_ENCODING_RAW,
		Value:        "hello world",
	}

	keys := make([]string, b.N)

	for i := 0; i < b.N; i++ {
		k := strconv.Itoa(i)
		keys[i] = k
		store[k] = obj
	}

	b.ResetTimer()

	var sink Obj
	var ok bool

	for i := 0; i < b.N; i++ {
		sink, ok = Get(keys[i])
	}

	_ = sink
	_ = ok
}
