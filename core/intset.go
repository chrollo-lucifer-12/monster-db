package core

import (
	"encoding/binary"
)

// int16 - 2 bytes

type Intset struct {
	length int32
	data   []byte
}

func NewIntset() *Intset {
	return &Intset{
		length: 0,
		data:   []byte{},
	}
}

func (is *Intset) search(val int16) int {
	l := 0
	r := int(is.length - 1)

	idx := -1

	for r >= l {
		mid := (l + r) / 2

		if is.get(mid) <= val {
			idx = mid
			l = mid + 1
		} else {
			r = mid - 1
		}
	}

	return idx
}

func (is *Intset) set(val int16) bool {

	idx := is.search(val)

	if idx != -1 && is.get(idx) == val {
		return false
	}

	insertAt := (idx + 1) * 2

	is.data = append(is.data, make([]byte, 2)...)

	copy(is.data[insertAt+2:], is.data[insertAt:])

	binary.LittleEndian.PutUint16(is.data[insertAt:], uint16(val))

	is.length++

	return true
}

func (is *Intset) get(idx int) int16 {
	offset := idx * 2
	return int16(binary.LittleEndian.Uint16(is.data[offset:]))
}
