package core

type Zset struct {
	dict map[string]*SkiplistNode
	list *Skiplist
}

func NewZset() *Zset {
	return &Zset{
		dict: make(map[string]*SkiplistNode),
		list: NewSkiplist(),
	}
}

func (z *Zset) Add(member string, score int) {

	z.Delete(member)

	insertedNode := z.list.insert(member, score)
	z.dict[member] = insertedNode
}

func (z *Zset) Delete(member string) int {
	foundNode, exists := z.dict[member]
	if exists {
		z.list.delete(member, foundNode.score)
		return 1
	}
	return 0
}

func (z *Zset) Search(member string) (int, bool) {
	foundNode, exists := z.dict[member]
	if exists {
		return foundNode.score, true
	}

	return 0, false
}

func (z *Zset) Range(start, stop int) []string {
	nodes := z.list.zrange(start, stop)

	res := make([]string, 0, len(nodes))
	for _, node := range nodes {
		res = append(res, node.member)
	}

	return res
}
