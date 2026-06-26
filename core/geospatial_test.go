package core

import (
	"context"
	"strconv"
	"testing"
)

var geoCtx = context.Background()

func BenchmarkGeoAddCmdExecute(b *testing.B) {
	store = make(map[string]Obj, 1)
	cmd := GeoAddCmd{}
	client := &benchClient{buf: make([]byte, 0, 4096)}

	key := "geo-key"

	g := NewGeoSpatial()
	Put(key, NewObj(g, OBJ_TYPE_GEO_SPATIAL, OBJ_ENCODING_TRIE), -1)

	args := make([]string, 1+3)
	args[0] = key

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client.buf = client.buf[:0]

		args[1] = strconv.Itoa(i % 180)
		args[2] = strconv.Itoa(i % 90)
		args[3] = "m" + strconv.Itoa(i)

		cmd.Execute(geoCtx, client, args)
	}
}

func BenchmarkGeoSearchCmdExecute(b *testing.B) {
	store = make(map[string]Obj, 1)

	cmd := GeoSearchCmd{}
	client := &benchClient{buf: make([]byte, 0, 32*1024)}

	key := "geo-key"
	g := NewGeoSpatial()
	Put(key, NewObj(g, OBJ_TYPE_GEO_SPATIAL, OBJ_ENCODING_TRIE), -1)

	for i := 0; i < 1000; i++ {
		g.Insert("m"+strconv.Itoa(i), float64(i%90), float64(i%180))
	}

	args := []string{
		key,
		"FROMLONLAT",
		"10",
		"10",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client.buf = client.buf[:0]
		cmd.Execute(geoCtx, client, args)
	}
}

func BenchmarkGeoHashCmdExecute(b *testing.B) {
	store = make(map[string]Obj, 1)

	cmd := GeoHashCmd{}
	client := &benchClient{buf: make([]byte, 0, 32*1024)}

	key := "geo-key"
	g := NewGeoSpatial()
	Put(key, NewObj(g, OBJ_TYPE_GEO_SPATIAL, OBJ_ENCODING_TRIE), -1)

	for i := 0; i < 1000; i++ {
		g.Insert("m"+strconv.Itoa(i), float64(i%90), float64(i%180))
	}

	args := []string{key, "m1", "m2", "m3"}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cmd.Execute(geoCtx, client, args)
	}
}
