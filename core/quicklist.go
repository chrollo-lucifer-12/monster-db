package core

const maxListpackBytes = 128

type QuicklistNode struct {
	lp   *Listpack
	prev *QuicklistNode
	next *QuicklistNode
}

type Quicklist struct {
	head *QuicklistNode
	tail *QuicklistNode
	len  int
}

func NewQuicklist() *Quicklist {
	lp := NewListPack()

	node := &QuicklistNode{
		lp: lp,
	}

	return &Quicklist{
		head: node,
		tail: node,
		len:  0,
	}
}

func add(node *QuicklistNode, val any, prepend bool) {
	switch v := val.(type) {
	case int:

		node.lp.AddInt(int64(v), prepend)

	case int64:

		node.lp.AddInt(v, prepend)

	case string:

		node.lp.AddString(v, prepend)

	default:
		panic("unsupported type")
	}
}

func (ql *Quicklist) addToTail(val any) {
	if ql.tail == nil {
		node := &QuicklistNode{lp: NewListPack()}
		ql.head = node
		ql.tail = node
	}

	if ql.tail.lp.totalLen() >= maxListpackBytes {
		newNode := &QuicklistNode{lp: NewListPack()}

		ql.tail.next = newNode
		newNode.prev = ql.tail

		ql.tail = newNode
	}

	add(ql.tail, val, false)

	ql.len++
}

func (ql *Quicklist) addToHead(val any) {
	if ql.head == nil {
		node := &QuicklistNode{lp: NewListPack()}
		ql.head = node
		ql.tail = node
	}

	if ql.head.lp.totalLen() >= maxListpackBytes {
		newNode := &QuicklistNode{lp: NewListPack()}

		newNode.next = ql.head
		ql.head.prev = newNode

		ql.head = newNode
	}

	add(ql.head, val, true)

	ql.len++
}

func (ql *Quicklist) GetElements(start, stop int) []any {
	if ql.head == nil || ql.len == 0 {
		return nil
	}

	var result []any
	index := 0

	for node := ql.head; node != nil; node = node.next {
		lp := node.lp
		pos := headerSize

		for pos < len(lp.data) && lp.data[pos] != endByte {
			val, size := lp.decodeAt(pos)

			if size <= 0 {
				return result
			}

			if val == nil {
				break
			}

			if index >= start && index <= stop {
				result = append(result, val)
			}

			pos += size
			index++

			if index > stop {
				return result
			}

		}

	}

	return result
}
