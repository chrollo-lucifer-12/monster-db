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

func (sl *Skiplist) delete(member string, score int) {
	current := sl.head

	updates := make([]*SkiplistNode, MAXLVL+1)

	for i := sl.level; i >= 0; i-- {
		for current.forward[i] != nil && (current.forward[i].score < score || (current.forward[i].score == score && current.forward[i].member < member)) {
			current = current.forward[i]
		}
		updates[i] = current
	}

	current = current.forward[0]

	if current != nil && current.member == member && current.score == score {
		for i := 0; i <= sl.level; i++ {
			if updates[i].forward[i] != current {
				continue
			}

			updates[i].forward[i] = current.forward[i]
		}

		for sl.level > 0 && sl.head.forward[sl.level] == nil {
			sl.level--
		}
	}
}

func (sl *Skiplist) search(member string, score int) *SkiplistNode {
	current := sl.head

	for i := sl.level; i >= 0; i-- {
		for current.forward[i] != nil && (current.forward[i].score < score || (current.forward[i].score == score && current.forward[i].member < member)) {
			current = current.forward[i]
		}
	}

	current = current.forward[0]

	if current != nil &&
		current.score == score &&
		current.member == member {
		return current
	}

	return nil
}

func (sl *Skiplist) zrange(start, stop int) []*SkiplistNode {
	result := []*SkiplistNode{}

	current := sl.head.forward[0]
	index := 0

	for current != nil && index < start {
		current = current.forward[0]
		index++
	}

	for current != nil && index <= stop {
		result = append(result, current)
		current = current.forward[0]
		index++
	}

	return result
}
