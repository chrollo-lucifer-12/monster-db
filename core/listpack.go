package core

import (
	"encoding/binary"
	"math"
)

const (
	endByte = 0xFF

	headerSize = 6
)

type Listpack struct {
	data []byte
}

func NewListPack() *Listpack {
	lp := &Listpack{
		data: make([]byte, headerSize+1),
	}

	lp.data[len(lp.data)-1] = endByte
	lp.setTotalLen(headerSize + 1)
	lp.setElements(0)

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

func (lp *Listpack) AddInt(val int64) {
	enc := encodeInt(val)
	lp.append(enc)
	lp.setElements(lp.elements() + 1)
}

func (lp *Listpack) AddString(s string) {
	enc := encodeString([]byte(s))
	lp.append(enc)
	lp.setElements(lp.elements() + 1)
}

func (lp *Listpack) append(entry []byte) {

	lp.data = lp.data[:len(lp.data)-1]

	lp.data = append(lp.data, entry...)

	lp.data = append(lp.data, endByte)

	lp.setTotalLen(len(lp.data))
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

func (lp *Listpack) Scan() []any {
	var res []any

	i := headerSize

	for i < len(lp.data)-1 {
		val, size := decodeEntry(lp.data[i:])
		res = append(res, val)
		i += size
	}

	return res
}

func decodeEntry(b []byte) (any, int) {
	if len(b) == 0 {
		return nil, 0
	}

	enc := b[0]

	if enc < 0x80 {
		return int64(enc), 1
	}

	if enc&0xC0 == 0x80 {
		val := int(enc&0x3F)<<8 | int(b[1])
		return int64(val), 2
	}

	if enc == 0xC0 {
		val := int16(binary.LittleEndian.Uint16(b[1:3]))
		return int64(val), 3
	}

	if enc == 0xD0 {
		val := int32(binary.LittleEndian.Uint32(b[1:5]))
		return int64(val), 5
	}

	if enc == 0xE0 {
		val := int64(binary.LittleEndian.Uint64(b[1:9]))
		return val, 9
	}

	if enc&0xC0 == 0x00 {
		n := int(enc & 0x3F)
		return string(b[1 : 1+n]), 1 + n
	}

	if enc&0xC0 == 0x40 {
		n := int(enc&0x3F)<<8 | int(b[1])
		return string(b[2 : 2+n]), 2 + n
	}

	if enc == 0x80 {
		n := int(binary.LittleEndian.Uint32(b[1:5]))
		return string(b[5 : 5+n]), 5 + n
	}

	return nil, 1
}
