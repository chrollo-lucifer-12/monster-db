package core

import (
	"encoding/binary"
	"math"
)

const (
	endByte = 0xFF

	headerSize = 6
)

const (
	TYPE_INT    = 0x01
	TYPE_STRING = 0x02
)

type Listpack struct {
	data []byte
}

func NewListPack() *Listpack {
	lp := &Listpack{
		data: make([]byte, headerSize+1),
	}

	lp.data[headerSize] = endByte
	lp.setMeta(0)

	return lp
}

func (lp *Listpack) totalLen() int {
	return int(binary.LittleEndian.Uint32(lp.data[0:4]))
}

func (lp *Listpack) setTotalLen(n int) {
	binary.LittleEndian.PutUint32(lp.data[0:4], uint32(n))
}

func (lp *Listpack) elements() int {
	return int(binary.LittleEndian.Uint16(lp.data[4:6]))
}

func (lp *Listpack) setElements(n int) {
	binary.LittleEndian.PutUint16(lp.data[4:6], uint16(n))
}

func (lp *Listpack) setMeta(n int) {
	lp.setTotalLen(len(lp.data))
	lp.setElements(lp.elements() + n)
}

func (lp *Listpack) AddInt(val int64, prepend bool) {
	enc := encodeInt(val)
	if prepend {
		lp.prepend(enc)
	} else {
		lp.append(enc)
	}

}

func (lp *Listpack) insert(pos int, entry []byte) {
	oldLen := len(lp.data)
	addLen := len(entry)

	lp.data = append(lp.data, make([]byte, addLen)...)

	copy(lp.data[pos+addLen:], lp.data[pos:oldLen])
	copy(lp.data[pos:], entry)

	lp.setMeta(1)
}

func (lp *Listpack) tail() int {
	return len(lp.data) - 1
}

func (lp *Listpack) append(entry []byte) {
	lp.insert(len(lp.data)-1, entry)
}

func (lp *Listpack) prepend(entry []byte) {
	lp.insert(headerSize, entry)
}

func (lp *Listpack) AddString(s string, prepend bool) {
	enc := encodeString([]byte(s))
	if prepend {
		lp.prepend(enc)
	} else {
		lp.append(enc)
	}
}

func (lp *Listpack) decodeAt(pos int) (any, int) {
	if pos >= len(lp.data) || lp.data[pos] == endByte {
		return nil, 0
	}

	b := lp.data[pos]

	if b < 0x80 {
		return int64(b), 1
	}

	if b&0xC0 == 0x80 {
		if pos+1 >= len(lp.data) {
			return nil, 0
		}
		v := int64(b&0x3F)<<8 | int64(lp.data[pos+1])
		return v, 2
	}

	if b == 0xC0 {
		if pos+3 > len(lp.data) {
			return nil, 0
		}
		v := int64(int16(binary.LittleEndian.Uint16(lp.data[pos+1 : pos+3])))
		return v, 3
	}

	if b == 0xD0 {
		if pos+5 > len(lp.data) {
			return nil, 0
		}
		v := int64(int32(binary.LittleEndian.Uint32(lp.data[pos+1 : pos+5])))
		return v, 5
	}

	if b == 0xE0 {
		if pos+9 > len(lp.data) {
			return nil, 0
		}
		v := int64(binary.LittleEndian.Uint64(lp.data[pos+1 : pos+9]))
		return v, 9
	}

	if b < 0x40 {
		length := int(b & 0x3F)
		if pos+1+length > len(lp.data) {
			return nil, 0
		}
		return string(lp.data[pos+1 : pos+1+length]), 1 + length
	}

	if b < 0x80 {
		if pos+2 > len(lp.data) {
			return nil, 0
		}
		length := int(b&0x3F)<<8 | int(lp.data[pos+1])
		if pos+2+length > len(lp.data) {
			return nil, 0
		}
		return string(lp.data[pos+2 : pos+2+length]), 2 + length
	}

	if pos+5 > len(lp.data) {
		return nil, 0
	}

	length := int(binary.LittleEndian.Uint32(lp.data[pos+1 : pos+5]))

	if pos+5+length > len(lp.data) {
		return nil, 0
	}

	return string(lp.data[pos+5 : pos+5+length]), 5 + length
}
func encodeInt(v int64) []byte {

	if v >= 0 && v < 128 {
		return []byte{byte(v)}
	}

	if v >= 0 && v < (1<<13) {
		b := make([]byte, 2)
		b[0] = 0x80 | byte(v>>8)
		b[1] = byte(v)
		return b
	}

	if v >= math.MinInt16 && v <= math.MaxInt16 {
		b := make([]byte, 3)
		b[0] = 0xC0
		binary.LittleEndian.PutUint16(b[1:], uint16(v))
		return b
	}

	if v >= math.MinInt32 && v <= math.MaxInt32 {
		b := make([]byte, 5)
		b[0] = 0xD0
		binary.LittleEndian.PutUint32(b[1:], uint32(v))
		return b
	}

	b := make([]byte, 9)
	b[0] = 0xE0
	binary.LittleEndian.PutUint64(b[1:], uint64(v))
	return b
}

func encodeString(s []byte) []byte {
	n := len(s)

	if n < 64 {
		out := make([]byte, 1+n)
		out[0] = byte(n & 0x3F) // keep last 6 bits
		copy(out[1:], s)
		return out
	}

	if n < 16384 {
		out := make([]byte, 2+n)
		out[0] = 0x40 | byte(n>>8) // marker + 6 high bits
		out[1] = byte(n)           // low 8 bits
		copy(out[2:], s)
		return out
	}

	out := make([]byte, 5+n)
	out[0] = 0x80
	binary.LittleEndian.PutUint32(out[1:], uint32(n))
	copy(out[5:], s)
	return out
}
