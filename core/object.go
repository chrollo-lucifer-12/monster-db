package core

import "unsafe"

type Obj struct {
	TypeEncoding   uint8
	Value          interface{}
	LastAccessedAt uint32
}

func (o *Obj) Size() int64 {
	size := int64(unsafe.Sizeof(*o))

	switch v := o.Value.(type) {
	case string:
		size += int64(len(v))

	case []byte:
		size += int64(len(v))
	}

	return size
}

var OBJ_TYPE_STRING uint8 = 0 << 4

var OBJ_ENCODING_RAW uint8 = 0
var OBJ_ENCODING_INT uint8 = 1
var OBJ_ENCODING_EMBSTR uint8 = 8
