package core

import "github.com/redis-server/internal/ds"

type Zset struct {
	dict map[string]*ds.SkiplistNode
	list *ds.Skiplist
}

func NewZset() *Zset {
	return &Zset{
		dict: make(map[string]*ds.SkiplistNode),
		list: ds.NewSkiplist(),
	}
}

func (z *Zset) Add(member string, score int) {

	z.Delete(member)

	insertedNode := z.list.Insert(member, score)
	z.dict[member] = insertedNode
}

func (z *Zset) Delete(member string) int {
	foundNode, exists := z.dict[member]
	if exists {
		z.list.Delete(member, foundNode.GetScore())
		return 1
	}
	return 0
}

func (z *Zset) Search(member string) (int, bool) {
	foundNode, exists := z.dict[member]
	if exists {
		return foundNode.GetScore(), true
	}

	return 0, false
}

func (z *Zset) Range(start, stop int) []string {
	nodes := z.list.Zrange(start, stop)

	res := make([]string, 0, len(nodes))
	for _, node := range nodes {
		res = append(res, node.GetMember())
	}

	return res
}
