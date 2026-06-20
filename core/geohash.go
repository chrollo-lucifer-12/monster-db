package core

import "strings"

type GeoEntry struct {
	Member string
	Lat    float64
	Lon    float64
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

func bitsToString(bits []int) string {
	var sb strings.Builder
	for _, b := range bits {
		if b == 0 {
			sb.WriteByte('0')
		} else {
			sb.WriteByte('1')
		}
	}
	return sb.String()
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
