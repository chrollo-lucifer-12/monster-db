package server

import "errors"

var (
	ErrOOM = errors.New("OOM command not allowed when used memory > 'maxmemory'")
)

type ZAllocator struct {
	usedMemory int64
	maxMemory  int64
}

func NewZAllocator(maxBytes int64) *ZAllocator {
	return &ZAllocator{
		usedMemory: 0,
		maxMemory:  maxBytes,
	}
}

func (z *ZAllocator) Alloc(size int) ([]byte, error) {

	current := z.usedMemory

	if z.maxMemory > 0 && current+int64(size) > z.maxMemory {
		return nil, ErrOOM
	}

	z.usedMemory += int64(size)

	return make([]byte, size), nil
}

func (z *ZAllocator) Free(b []byte) {
	cap := int64(cap(b))

	if cap == 0 {
		return
	}

	z.usedMemory -= cap
}
