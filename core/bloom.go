package core

import (
	"math"

	"github.com/spaolacci/murmur3"
)

func hashes(key string) (uint32, uint32) {
	data := []byte(key)

	h1 := murmur3.Sum32(data)
	h2 := murmur3.Sum32WithSeed(data, 12345)

	if h2 == 0 {
		h2 = 1
	}

	return h1, h2
}

type BloomFilter struct {
	filter []uint64
	size   int
	k      int
}

func NewBloomFilter(size int, errorRate float64) *BloomFilter {

	m := int(math.Ceil(-float64(size) * math.Log(errorRate) / (math.Ln2 * math.Ln2)))

	k := int(math.Ceil(
		(float64(m) / float64(size)) * math.Ln2,
	))

	return &BloomFilter{
		filter: make([]uint64, (m+63)/64),
		size:   m,
		k:      k,
	}
}

func (b *BloomFilter) Add(key string) {
	h1, h2 := hashes(key)

	for i := 0; i < b.k; i++ {
		idx := int(uint64(h1)+uint64(i)*uint64(h2)) % b.size

		word := idx / 64
		bit := idx % 64

		b.filter[word] |= uint64(1) << uint64(bit)
	}
}

func (b *BloomFilter) Exists(key string) bool {
	h1, h2 := hashes(key)

	for i := 0; i < b.k; i++ {
		idx := int(uint64(h1)+uint64(i)*uint64(h2)) % b.size

		word := idx / 64
		bit := idx % 64

		if b.filter[word]&(uint64(1)<<uint64(bit)) == 0 {
			return false
		}
	}

	return true
}
