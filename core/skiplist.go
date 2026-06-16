package core

import (
	"math/rand"
	"time"
)

const (
	P      = 0.5
	MAXLVL = 16
)

type SkiplistNode struct {
	key     int
	forward []*SkiplistNode
}

func NewSkipListNode(key, level int) *SkiplistNode {
	return &SkiplistNode{
		key:     key,
		forward: make([]*SkiplistNode, level+1),
	}
}

type Skiplist struct {
	level int
	head  *SkiplistNode
}

func NewSkiplist() *Skiplist {
	return &Skiplist{
		level: 0,
		head:  NewSkipListNode(-1, MAXLVL),
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randomLevel() int {

	r := rand.Float64()
	lvl := 0

	for r < P && lvl < MAXLVL {
		lvl++
		r = rand.Float64()
	}

	return lvl
}

func (sl *Skiplist) Insert(key int) {
	current := sl.head

}
