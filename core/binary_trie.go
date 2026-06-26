package core

type TrieNode struct {
	zero *TrieNode
	one  *TrieNode

	members []string
}

func NewTrieNode(zero *TrieNode, one *TrieNode) *TrieNode {
	return &TrieNode{
		zero:    zero,
		one:     one,
		members: make([]string, 0),
	}
}

type Trie struct {
	root *TrieNode
}

func NewTrie() *Trie {
	return &Trie{
		root: NewTrieNode(nil, nil),
	}
}

func (t *Trie) search(bits []uint64) []string {
	curr := t.root

	for _, b := range bits {
		if b == 0 {
			if curr.zero == nil {
				return nil
			}
			curr = curr.zero
		} else {
			if curr.one == nil {
				return nil
			}
			curr = curr.one
		}
	}

	return curr.members
}

func (t *Trie) insert(entry GeoEntry, bits []uint64) {
	curr := t.root

	for _, b := range bits {
		if b == 0 {
			if curr.zero == nil {
				curr.zero = &TrieNode{}
			}
			curr = curr.zero
		} else {
			if curr.one == nil {
				curr.one = &TrieNode{}
			}
			curr = curr.one
		}
	}

	curr.members = append(curr.members, entry.Member)
}
