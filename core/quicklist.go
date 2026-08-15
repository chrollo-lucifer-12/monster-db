package core

const maxListpackBytes = 8 * 1024

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

func (ql *Quicklist) splitTail() {
	newNode := &QuicklistNode{
		lp: NewListPack(),
	}

	ql.tail.next = newNode
	newNode.prev = ql.tail
	ql.tail = newNode
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

	estimated := 16
	if ql.tail.lp.totalLen()+estimated > maxListpackBytes {
		ql.splitTail()
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

	estimated := 16

	if ql.head.lp.totalLen()+estimated >= maxListpackBytes {
		newNode := &QuicklistNode{lp: NewListPack()}

		newNode.next = ql.head
		ql.head.prev = newNode

		ql.head = newNode
	}

	add(ql.head, val, true)

	ql.len++
}

func (ql *Quicklist) RemoveElements(count int, c ClientCommander) {
	if ql.head == nil || ql.len == 0 || count <= 0 {
		c.AppendNull()
		return
	}

	if count > ql.len {
		count = ql.len
	}

	c.AppendArrayLen(count)

	removed := 0
	node := ql.head

	for node != nil && removed < count {
		lp := node.lp
		next := node.next

		removeStart := headerSize
		totalSize := 0
		removedInNode := 0

		pos := headerSize

		for pos < len(lp.data) &&
			lp.data[pos] != endByte &&
			removed < count {

			t := lp.data[pos]

			var size int

			switch t {
			case TYPE_STRING:
				val, s := lp.decodeAtString(pos)
				if s <= 0 {
					return
				}

				size = s
				c.AppendBytesReply(val)

			case TYPE_INT:
				val, s := lp.decodeAtInt(pos)
				if s <= 0 {
					return
				}

				size = s
				c.AppendIntReply(val)

			default:
				return
			}

			totalSize += size
			pos += size

			removed++
			removedInNode++
		}

		if totalSize > 0 {
			lp.remove(removeStart, totalSize)

			lp.setElements(lp.elements() - removedInNode)
			lp.setTotalLen(len(lp.data))

			ql.len -= removedInNode
		}

		if lp.IsEmpty() {
			if node.prev != nil {
				node.prev.next = next
			} else {
				ql.head = next
			}

			if next != nil {
				next.prev = node.prev
			} else {
				ql.tail = node.prev
			}
		}

		node = next
	}

	if ql.head == nil {
		ql.tail = nil
	}
}

func (ql *Quicklist) GetElements(start, stop int, c ClientCommander) {
	if ql.head == nil || ql.len == 0 {
		c.AppendNull()
		return
	}

	c.AppendArrayLen(stop - start + 1)
	index := 0

	for node := ql.head; node != nil; node = node.next {
		lp := node.lp
		pos := headerSize

		for pos < len(lp.data) && lp.data[pos] != endByte {

			t := lp.data[pos]

			var size int

			if t == TYPE_STRING {
				val, s := lp.decodeAtString(pos)

				if s <= 0 {
					return
				}
				size = s

				if index >= start && index <= stop {
					c.AppendBytesReply(val)
				}
			} else if t == TYPE_INT {
				val, s := lp.decodeAtInt(pos)

				if s <= 0 {
					return
				}
				size = s

				if index >= start && index <= stop {
					c.AppendIntReply(val)
				}
			}

			pos += size
			index++

			if index > stop {
				return
			}

		}

	}

}
