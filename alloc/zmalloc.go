package alloc

import (
	"errors"
	"sync"
	"sync/atomic"
)

var (
	ErrOOM          = errors.New("OOM command not allowed when used memory > 'maxmemory'")
	globalAllocator *ZAllocator
	initOnce        sync.Once
)

type ZAllocator struct {
	usedMemory atomic.Int64
	maxMemory  int64
}

func InitGlobalAllocator(maxBytes int64) {
	initOnce.Do(func() {
		globalAllocator = &ZAllocator{
			maxMemory: maxBytes,
		}
	})
}

func NewZAllocator(maxBytes int64) *ZAllocator {
	return &ZAllocator{
		maxMemory: maxBytes,
	}
}

func Alloc(size int) ([]byte, error) {
	if size <= 0 {
		return nil, nil
	}

	if globalAllocator.maxMemory > 0 && globalAllocator.usedMemory.Load()+int64(size) > globalAllocator.maxMemory {
		return nil, ErrOOM
	}

	globalAllocator.usedMemory.Add(int64(size))

	return make([]byte, size), nil
}

func Free(b []byte) {
	allocatedSize := int64(cap(b))

	if allocatedSize == 0 {
		return
	}

	globalAllocator.usedMemory.Add(-allocatedSize)
}

func UsedMemory() int64 {
	return globalAllocator.usedMemory.Load()
}
