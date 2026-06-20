package core

import (
	"context"
	"strconv"

	"github.com/redis-server/resp"
)

type GeoAddCmd struct{}

type GeoHashCmd struct{}

func (GeoAddCmd) Name() string { return "GEOADD" }

func (GeoHashCmd) Name() string { return "GEOHASH" }

func (GeoHashCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) < 2 {
		return errWrongArgs("geohash")
	}

	key := args[0]
	obj := Get(key)

	if obj == nil {
		return RESP_NIL
	}

	if err := assertType(obj.TypeEncoding, OBJ_TYPE_GEO_SPATIAL); err != nil {
		return errWrongType()
	}

	g := obj.Value.(*GeoSpatial)

	hashes := g.GeoHash(args[1:])

	return resp.Encode(hashes, false)
}

func (GeoAddCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) < 4 {
		return errWrongArgs("geoadd")
	}

	key := args[0]
	obj := Get(key)

	var g *GeoSpatial

	if obj == nil {
		g = NewGeoSpatial()
		Put(key, NewObj(g, -1, OBJ_TYPE_GEO_SPATIAL, OBJ_ENCODING_TRIE))
	} else {
		g = obj.Value.(*GeoSpatial)
	}

	i := 1
	count := 0

	for ; i < len(args); i += 3 {
		lon, _ := strconv.ParseFloat(args[i], 64)
		lat, _ := strconv.ParseFloat(args[i+1], 64)
		member := args[i+2]
		g.Insert(member, lat, lon)
		count++
	}

	return resp.Encode(count, false)
}
