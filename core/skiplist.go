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
	member  string
	score   int
	forward []*SkiplistNode
}

func NewSkipListNode(member string, score int, level int) *SkiplistNode {
	return &SkiplistNode{
		member:  member,
		score:   score,
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
		head:  NewSkipListNode("root", -1, MAXLVL),
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

func (sl *Skiplist) insert(member string, score int) *SkiplistNode {
	current := sl.head

	updates := make([]*SkiplistNode, MAXLVL+1)

	for i := sl.level; i >= 0; i-- {
		for current.forward[i] != nil && current.forward[i].score < score {
			current = current.forward[i]
		}
		updates[i] = current
	}

	current = current.forward[0]

	// if current != nil && current.key == key {
	// 	return
	// }

	lvl := randomLevel()

	if lvl > sl.level {
		for i := sl.level + 1; i <= lvl; i++ {
			updates[i] = sl.head
		}
		sl.level = lvl
	}

	newNode := NewSkipListNode(member, score, lvl)

	for i := 0; i <= lvl; i++ {
		newNode.forward[i] = updates[i].forward[i]
		updates[i].forward[i] = newNode
	}

	return newNode
}
