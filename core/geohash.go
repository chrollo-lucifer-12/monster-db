package core

type GeoEntry struct {
	Member string
	Lat    float64
	Lon    float64
}

type TrieNode struct {
	zero *TrieNode
	one  *TrieNode

	members []GeoEntry
}

func NewTrieNode(zero *TrieNode, one *TrieNode) *TrieNode {
	return &TrieNode{
		zero:    zero,
		one:     one,
		members: make([]GeoEntry, 0),
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

func (t *Trie) insert(entry GeoEntry, bits []int) {
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

	curr.members = append(curr.members, entry)
}

func geoHash(lat, lon float64, bits int) []int {
	latRange := []float64{-90, 90}
	lonRange := []float64{-180, 180}

	res := make([]int, 0, bits)
	isLon := true

	for i := 0; i < bits; i++ {
		if isLon {
			mid := (lonRange[0] + lonRange[1]) / 2

			if lon > mid {
				res = append(res, 1)
				lonRange[0] = mid
			} else {
				res = append(res, 0)
				lonRange[1] = mid
			}
		} else {
			mid := (latRange[0] + latRange[1]) / 2
			if lat > mid {
				res = append(res, 1)
				latRange[0] = mid
			} else {
				res = append(res, 0)
				latRange[1] = mid
			}
		}

		isLon = !isLon
	}

	return res
}
