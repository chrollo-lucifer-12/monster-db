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

func (z *Zset) Delete(member string) {
	foundNode, exists := z.dict[member]
	if exists {
		z.list.delete(member, foundNode.score)
	}
}

func (z *Zset) Search(member string) (int, bool) {
	foundNode, exists := z.dict[member]
	if exists {
		return foundNode.score, true
	}

	return 0, false
}
