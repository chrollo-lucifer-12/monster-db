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
		data: make([]byte, headerSize+1, maxListpackBytes),
	}

	lp.data[headerSize] = endByte
	lp.setMeta(0)

	return lp
}

func (lp *Listpack) IsEmpty() bool {
	return len(lp.data) <= headerSize+1 || lp.elements() == 0
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
	var entryLen int
	switch {
	case val >= 0 && val < 128:
		entryLen = 2
	case val >= 0 && val < (1<<13):
		entryLen = 3
	case val >= math.MinInt16 && val <= math.MaxInt16:
		entryLen = 4
	case val >= math.MinInt32 && val <= math.MaxInt32:
		entryLen = 6
	default:
		entryLen = 10
	}

	pos := lp.tail()
	if prepend {
		pos = headerSize
	}

	lp.grow(pos, entryLen)

	lp.data[pos] = TYPE_INT
	switch entryLen {
	case 2:
		lp.data[pos+1] = byte(val)
	case 3:
		lp.data[pos+1] = 0x80 | byte(val>>8)
		lp.data[pos+2] = byte(val)
	case 4:
		lp.data[pos+1] = 0xC0
		binary.LittleEndian.PutUint16(lp.data[pos+2:], uint16(val))
	case 6:
		lp.data[pos+1] = 0xD0
		binary.LittleEndian.PutUint32(lp.data[pos+2:], uint32(val))
	case 10:
		lp.data[pos+1] = 0xE0
		binary.LittleEndian.PutUint64(lp.data[pos+2:], uint64(val))
	}

	lp.setMeta(1)
}

func (lp *Listpack) remove(pos int, length int) {
	oldLen := len(lp.data)

	if lp.data == nil || pos < 0 || pos >= oldLen || length <= 0 {
		return
	}

	if pos+length > oldLen {
		length = oldLen - pos
	}

	copy(lp.data[pos:], lp.data[pos+length:oldLen])

	lp.data = lp.data[:oldLen-length]

	lp.setMeta(-1)
}

func (lp *Listpack) grow(pos, addLen int) {
	oldLen := len(lp.data)
	newLen := oldLen + addLen
	if cap(lp.data) < newLen {
		newData := make([]byte, newLen, newLen*2)
		copy(newData[:pos], lp.data[:pos])
		copy(newData[pos+addLen:], lp.data[pos:])
		lp.data = newData
	} else {
		lp.data = lp.data[:newLen]
		copy(lp.data[pos+addLen:], lp.data[pos:oldLen])
	}
}

func (lp *Listpack) insert(pos int, entry []byte) {
	oldLen := len(lp.data)
	addLen := len(entry)

	if cap(lp.data) < oldLen+addLen {
		newData := make([]byte, oldLen+addLen, (oldLen+addLen)*2)
		copy(newData, lp.data)
		lp.data = newData
	} else {
		lp.data = lp.data[:oldLen+addLen]
		copy(lp.data[pos+addLen:], lp.data[pos:oldLen])
	}
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
	n := len(s)
	var entryLen int
	if n < 64 {
		entryLen = 2 + n
	} else if n < 16384 {
		entryLen = 3 + n
	} else {
		entryLen = 6 + n
	}

	pos := lp.tail()
	if prepend {
		pos = headerSize
	}

	lp.grow(pos, entryLen)

	lp.data[pos] = TYPE_STRING
	if n < 64 {
		lp.data[pos+1] = byte(n & 0x3F)
		copy(lp.data[pos+2:], s)
	} else if n < 16384 {
		lp.data[pos+1] = 0x40 | byte(n>>8)
		lp.data[pos+2] = byte(n)
		copy(lp.data[pos+3:], s)
	} else {
		lp.data[pos+1] = 0x80
		binary.LittleEndian.PutUint32(lp.data[pos+2:], uint32(n))
		copy(lp.data[pos+6:], s)
	}

	lp.setMeta(1)
}

func (lp *Listpack) decodeAtString(pos int) ([]byte, int) {
	idx := pos + 1

	if idx >= len(lp.data) {
		return []byte{}, 0
	}
	b := lp.data[idx]

	if b < 0x40 {
		length := int(b & 0x3F)
		if idx+1+length > len(lp.data) {
			return []byte{}, 0
		}
		return (lp.data[idx+1 : idx+1+length]), 2 + length
	}
	if b < 0x80 {
		if idx+2 > len(lp.data) {
			return []byte{}, 0
		}
		length := int(b&0x3F)<<8 | int(lp.data[idx+1])
		if idx+2+length > len(lp.data) {
			return []byte{}, 0
		}
		return (lp.data[idx+2 : idx+2+length]), 3 + length
	}
	if idx+5 > len(lp.data) {
		return []byte{}, 0
	}
	length := int(binary.LittleEndian.Uint32(lp.data[idx+1 : idx+5]))
	if idx+5+length > len(lp.data) {
		return []byte{}, 0
	}
	return (lp.data[idx+5 : idx+5+length]), 6 + length
}

func (lp *Listpack) decodeAtInt(pos int) (int64, int) {
	idx := pos + 1

	if idx >= len(lp.data) {
		return 0, 0
	}
	b := lp.data[idx]

	if b < 0x80 {
		return int64(b), 2
	}
	if b&0xC0 == 0x80 {
		if idx+1 >= len(lp.data) {
			return 0, 0
		}
		v := int64(b&0x3F)<<8 | int64(lp.data[idx+1])
		return v, 3
	}
	if b == 0xC0 {
		if idx+3 > len(lp.data) {
			return 0, 0
		}
		v := int64(int16(binary.LittleEndian.Uint16(lp.data[idx+1 : idx+3])))
		return v, 4
	}
	if b == 0xD0 {
		if idx+5 > len(lp.data) {
			return 0, 0
		}
		v := int64(int32(binary.LittleEndian.Uint32(lp.data[idx+1 : idx+5])))
		return v, 6
	}

	if idx+9 > len(lp.data) {
		return 0, 0
	}
	v := int64(binary.LittleEndian.Uint64(lp.data[idx+1 : idx+9]))
	return v, 10

}

func (lp *Listpack) decodeAt(pos int) (any, int) {
	if pos >= len(lp.data) || lp.data[pos] == endByte {
		return nil, 0
	}

	t := lp.data[pos]
	idx := pos + 1

	if t == TYPE_INT {
		if idx >= len(lp.data) {
			return nil, 0
		}
		b := lp.data[idx]

		if b < 0x80 {
			return int64(b), 2
		}
		if b&0xC0 == 0x80 {
			if idx+1 >= len(lp.data) {
				return nil, 0
			}
			v := int64(b&0x3F)<<8 | int64(lp.data[idx+1])
			return v, 3
		}
		if b == 0xC0 {
			if idx+3 > len(lp.data) {
				return nil, 0
			}
			v := int64(int16(binary.LittleEndian.Uint16(lp.data[idx+1 : idx+3])))
			return v, 4
		}
		if b == 0xD0 {
			if idx+5 > len(lp.data) {
				return nil, 0
			}
			v := int64(int32(binary.LittleEndian.Uint32(lp.data[idx+1 : idx+5])))
			return v, 6
		}
		if b == 0xE0 {
			if idx+9 > len(lp.data) {
				return nil, 0
			}
			v := int64(binary.LittleEndian.Uint64(lp.data[idx+1 : idx+9]))
			return v, 10
		}
	}

	if t == TYPE_STRING {
		if idx >= len(lp.data) {
			return nil, 0
		}
		b := lp.data[idx]

		if b < 0x40 {
			length := int(b & 0x3F)
			if idx+1+length > len(lp.data) {
				return nil, 0
			}
			return string(lp.data[idx+1 : idx+1+length]), 2 + length
		}
		if b < 0x80 {
			if idx+2 > len(lp.data) {
				return nil, 0
			}
			length := int(b&0x3F)<<8 | int(lp.data[idx+1])
			if idx+2+length > len(lp.data) {
				return nil, 0
			}
			return string(lp.data[idx+2 : idx+2+length]), 3 + length
		}
		if idx+5 > len(lp.data) {
			return nil, 0
		}
		length := int(binary.LittleEndian.Uint32(lp.data[idx+1 : idx+5]))
		if idx+5+length > len(lp.data) {
			return nil, 0
		}
		return string(lp.data[idx+5 : idx+5+length]), 6 + length
	}

	return nil, 0
}

func encodeInt(v int64) []byte {
	switch {
	case v >= 0 && v < 128:
		return []byte{TYPE_INT, byte(v)}

	case v >= 0 && v < (1<<13):
		out := make([]byte, 3)
		out[0] = TYPE_INT
		out[1] = 0x80 | byte(v>>8)
		out[2] = byte(v)
		return out

	case v >= math.MinInt16 && v <= math.MaxInt16:
		out := make([]byte, 4)
		out[0] = TYPE_INT
		out[1] = 0xC0
		binary.LittleEndian.PutUint16(out[2:], uint16(v))
		return out

	case v >= math.MinInt32 && v <= math.MaxInt32:
		out := make([]byte, 6)
		out[0] = TYPE_INT
		out[1] = 0xD0
		binary.LittleEndian.PutUint32(out[2:], uint32(v))
		return out

	default:
		out := make([]byte, 10)
		out[0] = TYPE_INT
		out[1] = 0xE0
		binary.LittleEndian.PutUint64(out[2:], uint64(v))
		return out
	}
}

func encodeString(s string) []byte {
	n := len(s)

	if n < 64 {
		b := make([]byte, 2+n)
		b[0] = TYPE_STRING
		b[1] = byte(n & 0x3F)
		copy(b[2:], s)
		return b
	} else if n < 16384 {
		b := make([]byte, 3+n)
		b[0] = TYPE_STRING
		b[1] = 0x40 | byte(n>>8)
		b[2] = byte(n)
		copy(b[3:], s)
		return b
	}
	b := make([]byte, 6+n)
	b[0] = TYPE_STRING
	b[1] = 0x80
	binary.LittleEndian.PutUint32(b[2:], uint32(n))
	copy(b[6:], s)
	return b
}
