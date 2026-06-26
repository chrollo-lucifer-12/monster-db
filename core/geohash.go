package core

import (
	"strconv"
)

type GeoEntry struct {
	Member string
	Lat    float64
	Lon    float64
}

func (e GeoEntry) String() []string {
	sLat := strconv.FormatFloat(e.Lat, 'f', -1, 64)
	sLon := strconv.FormatFloat(e.Lon, 'f', -1, 64)

	return []string{sLat, sLon}
}

type GeoSpatial struct {
	trie  *Trie
	index map[string]GeoEntry
}

func NewGeoSpatial() *GeoSpatial {
	return &GeoSpatial{
		trie:  NewTrie(),
		index: make(map[string]GeoEntry),
	}
}

func (g *GeoSpatial) Search(lat, lon float64) []string {
	bits := geoHash(lat, lon, 20)

	members := g.trie.search(bits)

	return members
}

func (g *GeoSpatial) Insert(member string, lat, lon float64) {
	entry := GeoEntry{
		Member: member,
		Lat:    lat,
		Lon:    lon,
	}

	bits := geoHash(lat, lon, 20)

	g.trie.insert(entry, bits)

	g.index[member] = entry
}

func (g *GeoSpatial) GeoHash(members []string) []string {

	var res []string

	for _, member := range members {
		entry, ok := g.index[member]
		if !ok {
			res = append(res, "nil")
			continue
		}

		res = append(res, bitsToString(geoHash(entry.Lat, entry.Lon, 20)))
	}

	return res
}

func bitsToString(bits []uint64) string {
	n := len(bits)
	b := make([]byte, 0, n)

	for _, v := range bits {
		if v == 0 {
			b = append(b, '0')
		} else {
			b = append(b, '1')
		}
	}

	return string(b)
}

func geoHash(lat, lon float64, bits int) []uint64 {
	latRange := [2]float64{-90, 90}
	lonRange := [2]float64{-180, 180}

	out := make([]uint64, (bits+63)/64)

	isLon := true

	for i := 0; i < bits; i++ {
		var bit uint64

		if isLon {
			mid := (lonRange[0] + lonRange[1]) / 2
			if lon > mid {
				bit = 1
				lonRange[0] = mid
			} else {
				bit = 0
				lonRange[1] = mid
			}
		} else {
			mid := (latRange[0] + latRange[1]) / 2
			if lat > mid {
				bit = 1
				latRange[0] = mid
			} else {
				bit = 0
				latRange[1] = mid
			}
		}

		word := i / 64
		offset := 63 - (i % 64)

		out[word] |= bit << offset
		isLon = !isLon
	}

	return out
}
