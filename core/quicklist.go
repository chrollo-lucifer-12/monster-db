package core

import "github.com/redis-server/resp"

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
	if ql.head == nil || ql.len == 0 {
		c.AppendReply(nil, false)
		return
	}

	c.AppendReply(resp.ArrayLen(count), false)
	removed := 0
	for node := ql.head; node != nil && removed < count; node = node.next {
		lp := node.lp
		pos := headerSize
		for pos < len(lp.data) && lp.data[pos] != endByte && removed < count {
			t := lp.data[pos]

			var size int

			if t == TYPE_STRING {
				val, s := lp.decodeAtString(pos)

				if s <= 0 {
					return
				}
				size = s
				// if val == "" {
				// 	break
				// }

				c.AppendBytesReply(val)

			} else if t == TYPE_INT {
				val, s := lp.decodeAtInt(pos)

				if s <= 0 {
					return
				}
				size = s
				// if val == "" {
				// 	break
				// }

				c.AppendIntReply(val)

			}

			lp.remove(headerSize, size)
			ql.len--
			removed++
		}

		if lp.IsEmpty() {
			ql.head = ql.head.next
			if ql.head != nil {
				ql.head.prev = nil
			}
		}
	}
}

func (ql *Quicklist) GetElements(start, stop int, c ClientCommander) {
	if ql.head == nil || ql.len == 0 {
		c.AppendReply(nil, false)
		return
	}

	c.AppendReply(resp.ArrayLen(stop-start+1), false)
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
				// if val == "" {
				// 	break
				// }

				if index >= start && index <= stop {
					c.AppendBytesReply(val)
				}
			} else if t == TYPE_INT {
				val, s := lp.decodeAtInt(pos)

				if s <= 0 {
					return
				}
				size = s
				// if val == "" {
				// 	break
				// }

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
