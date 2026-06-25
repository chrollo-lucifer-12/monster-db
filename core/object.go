package core

type Obj struct {
	TypeEncoding   uint8
	StrVal         string
	LastAccessedAt uint32
	IntVal         int64
	Value          interface{}
}

func NewStringObj(value string, oType uint8, oEnc uint8) Obj {
	return Obj{
		StrVal:         value,
		TypeEncoding:   oType | oEnc,
		LastAccessedAt: getCurrentClock(),
	}
}

func NewPtrObj(value interface{}, oType uint8, oEnc uint8) Obj {
	return Obj{
		Value:          value,
		TypeEncoding:   oType | oEnc,
		LastAccessedAt: getCurrentClock(),
	}
}

var OBJ_TYPE_STRING uint8 = 0 << 4
var OBJ_TYPE_LIST uint8 = 1 << 4
var OBJ_TYPE_BLOOM_FILTERS uint8 = 2 << 4
var OBJ_TYPE_SET uint8 = 3 << 4
var OBJ_TYPE_ZSET uint8 = 4 << 4
var OBJ_TYPE_GEO_SPATIAL uint8 = 5 << 4

var OBJ_ENCODING_RAW uint8 = 0
var OBJ_ENCODING_INT uint8 = 1
var OBJ_ENCODING_EMBSTR uint8 = 8
var OBJ_ENCODING_LISTPACK uint8 = 3
var OBJ_ENCODING_BOOL_ARR uint8 = 2
var OBJ_ENCODING_INSET uint8 = 4
var OBJ_ENCODING_SKIPLIST uint8 = 5
var OBJ_ENCODING_TRIE uint8 = 6
