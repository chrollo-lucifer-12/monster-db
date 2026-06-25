package core

import (
	"strconv"
	"testing"
)

func BenchmarkListpack_AddInt(b *testing.B) {
	lp := NewListPack()
	b.ResetTimer()

	for i := 0; i < 100000; i++ {
		lp.AddInt(int64(i), false)
	}
}

func BenchmarkListpack_AddString(b *testing.B) {
	lp := NewListPack()
	b.ResetTimer()

	for i := 0; i < 10000; i++ {
		lp.AddString(strconv.Itoa(i), false)
	}
}

func BenchmarkListpack_InsertRemove(b *testing.B) {
	for i := 0; i < b.N; i++ {
		lp := NewListPack()
		for j := 0; j < 1000; j++ {
			lp.AddInt(int64(j), false)
		}
		b.ResetTimer()
		lp.remove(headerSize, 2)
	}
}

func BenchmarkListpack_DecodeSequential(b *testing.B) {
	lp := NewListPack()
	for i := 0; i < 10000; i++ {
		lp.AddInt(int64(i), false)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pos := headerSize
		for pos < len(lp.data) && lp.data[pos] != endByte {
			_, size := lp.decodeAt(pos)
			if size <= 0 {
				break
			}
			pos += size
		}
	}
}

func BenchmarkListpack_MixedOps(b *testing.B) {
	lp := NewListPack()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			lp.AddInt(int64(i), false)
		} else {
			lp.AddString("hello", false)
		}
	}
}
