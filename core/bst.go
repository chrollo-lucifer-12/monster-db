package core

type BSTNode struct {
	Key   string
	Value any
	Left  *BSTNode
	Right *BSTNode
}

type BST struct {
	root *BSTNode
}

func (b *BST) Insert(key string, value int) {
	if b.root == nil {
		b.root = &BSTNode{Key: key, Value: value}
		return
	}

	b.root.insertNode(key, value)
}

func (n *BSTNode) insertNode(key string, value int) {
	if key < n.Key {
		if n.Left == nil {
			n.Left = &BSTNode{Key: key, Value: value}
		} else {
			n.Left.insertNode(key, value)
		}
	} else if key > n.Key {
		if n.Right == nil {
			n.Right = &BSTNode{Key: key, Value: value}
		} else {
			n.Right.insertNode(key, value)
		}
	} else {
		n.Value = value
	}
}

func (b *BST) Search(key string) (any, bool) {
	n := b.root
	for n != nil {
		if key < n.Key {
			n = n.Left
		} else if key > n.Key {
			n = n.Right
		} else {
			return n.Value, true
		}
	}
	return nil, false
}
