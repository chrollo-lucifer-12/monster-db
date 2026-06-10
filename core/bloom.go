package core

import (
	"hash"
	"time"

	"github.com/spaolacci/murmur3"
)

var hasher hash.Hash32

func murmurhash(key string, size int) int {
	hasher.Write([]byte(key))
	res := int(hasher.Sum32()) % size
	hasher.Reset()
	return res
}

type BloomFilter struct {
	filter []bool
	size   int
}

func NewBloomFilter(size int) *BloomFilter {

	hasher = murmur3.New32WithSeed(uint32(time.Now().Unix()))

	return &BloomFilter{
		filter: make([]bool, size),
		size:   size,
	}
}

func (b *BloomFilter) Add(key string) {
	idx := murmurhash(key, b.size)
	b.filter[idx] = true
}

func (b *BloomFilter) Exists(key string) bool {
	idx := murmurhash(key, b.size)
	return b.filter[idx]
}
