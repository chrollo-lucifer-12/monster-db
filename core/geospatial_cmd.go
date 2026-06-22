package core

import (
	"context"
	"strconv"

	"github.com/redis-server/resp"
)

type GeoAddCmd struct{}

type GeoHashCmd struct{}

type GeoDistCmd struct{}

type GeoPosCmd struct{}

type GeoSearchCmd struct{}

func (GeoSearchCmd) Name() string { return "GEOSEARCH" }

func (GeoAddCmd) Name() string { return "GEOADD" }

func (GeoHashCmd) Name() string { return "GEOHASH" }

func (GeoDistCmd) Name() string { return "GEODIST" }

func (GeoPosCmd) Name() string { return "GEOPOS" }

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

func (GeoDistCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) != 3 {
		return errWrongArgs("geodist")
	}

	key := args[0]
	member1 := args[1]
	member2 := args[2]

	obj := Get(key)

	if obj == nil {
		return RESP_NIL
	}

	if err := assertType(obj.TypeEncoding, OBJ_TYPE_GEO_SPATIAL); err != nil {
		return errWrongType()
	}

	g := obj.Value.(*GeoSpatial)

	e1, ok := g.index[member1]
	if !ok {
		return RESP_NIL
	}

	e2, ok := g.index[member2]
	if !ok {
		return RESP_NIL
	}

	dist := Haversine(e1.Lat, e1.Lon, e2.Lat, e2.Lon)

	return resp.Encode(dist, false)
}

func (GeoPosCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) < 2 {
		return errWrongArgs("geopos")
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

	var res []any

	for _, member := range args[1:] {
		e, ok := g.index[member]
		if !ok {
			res = append(res, nil)
			continue
		}

		res = append(res, e.String())
	}

	return resp.Encode(res, false)
}

func (GeoSearchCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	if len(args) != 4 {
		return errWrongArgs("geosearch")
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

	lat, _ := strconv.ParseFloat(args[2], 64)
	lon, _ := strconv.ParseFloat(args[3], 64)

	return resp.Encode(g.Search(lat, lon), false)
}
