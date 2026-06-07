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

func (lp *Listpack) AddInt(val int64, prepend bool) {
	enc := encodeInt(val)
	if prepend {
		lp.prepend(enc)
	} else {
		lp.append(enc)
	}
	lp.setElements(lp.elements() + 1)
}

func (lp *Listpack) AddString(s string, prepend bool) {
	enc := encodeString([]byte(s))
	if prepend {
		lp.prepend(enc)
	} else {
		lp.append(enc)
	}
	lp.setElements(lp.elements() + 1)
}

func (lp *Listpack) prepend(entry []byte) {
	newTotalSize := len(lp.data) + len(entry)
	newData := make([]byte, newTotalSize)

	copy(newData[:headerSize], lp.data[:headerSize])
	copy(newData[headerSize:], entry)
	copy(newData[headerSize+len(entry):], lp.data[headerSize:])

	lp.data = newData

	lp.setTotalLen(newTotalSize)
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
