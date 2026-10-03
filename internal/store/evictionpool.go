package core

import "sort"

type PoolItem struct {
	key            string
	lastAccessedAt uint32
}

type EvictionPool struct {
	pool   []*PoolItem
	keyset map[string]*PoolItem
}

type ByIdleTime []*PoolItem

func (a ByIdleTime) Len() int {
	return len(a)
}

func (a ByIdleTime) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByIdleTime) Less(i, j int) bool {
	if a[i] == nil || a[j] == nil {
		return false
	}
	return a[i].lastAccessedAt < a[j].lastAccessedAt
}

func (pq *EvictionPool) Push(key string, lastAccessedAt uint32) {
	_, ok := pq.keyset[key]

	if ok {
		return
	}

	item := &PoolItem{key: key, lastAccessedAt: lastAccessedAt}

	pq.pool = append(pq.pool, item)
	pq.keyset[key] = item

	sort.Sort(ByIdleTime(pq.pool))

	if len(pq.pool) > cap(pq.pool) {
		evicted := pq.pool[len(pq.pool)-1]
		pq.pool = pq.pool[:len(pq.pool)-1]
		delete(pq.keyset, evicted.key)
	}
}

func (pq *EvictionPool) Pop() *PoolItem {
	if len(pq.pool) == 0 {
		return nil
	}

	item := pq.pool[0]
	pq.pool = pq.pool[1:]

	return item
}

func newEvictionPool(size int) *EvictionPool {
	return &EvictionPool{
		pool:   make([]*PoolItem, 0, size),
		keyset: make(map[string]*PoolItem),
	}
}
