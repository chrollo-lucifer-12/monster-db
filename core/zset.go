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
	insertedNode := z.list.insert(member, score)
	z.dict[member] = insertedNode
}
