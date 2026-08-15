package core

import (
	"context"
	"strconv"
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

func (GeoHashCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) < 2 {
		c.AppendError(errWrongArgs("geohash"))
		return
	}

	key := args[0]
	obj, exists := Get(key)

	if !exists {
		c.AppendNull()
	}

	if err := assertType(obj.TypeEncoding, OBJ_TYPE_GEO_SPATIAL); err != nil {
		c.AppendError(errWrongType())
	}

	c.AppendStrArray(obj.Value.(*GeoSpatial).GeoHash(args[1:]))
}

func (GeoAddCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) < 4 {
		c.AppendError(errWrongArgs("geoadd"))
		//c.AppendReply(errWrongArgs("geoadd"), false)
		return
	}

	key := args[0]
	obj, exists := Get(key)

	var g *GeoSpatial

	if !exists {
		g = NewGeoSpatial()
		Put(key, NewObj(g, OBJ_TYPE_GEO_SPATIAL, OBJ_ENCODING_TRIE), -1)
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

	c.AppendIntReply(int64(count))
}

func (GeoDistCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 3 {
		c.AppendError(errWrongArgs("geodist"))
		//	c.AppendReply(errWrongArgs("geodist"), false)
		return
	}

	key := args[0]
	member1 := args[1]
	member2 := args[2]

	obj, exists := Get(key)

	if !exists {
		//	c.AppendReply(nil, false)
		c.AppendNull()
		return
	}

	if err := assertType(obj.TypeEncoding, OBJ_TYPE_GEO_SPATIAL); err != nil {
		c.AppendError(errWrongType())
		//c.AppendReply(errWrongType(), false)
		return
	}

	g := obj.Value.(*GeoSpatial)

	e1, ok := g.index[member1]
	if !ok {
		c.AppendNull()
		//c.AppendReply(nil, false)
		return
	}

	e2, ok := g.index[member2]
	if !ok {
		c.AppendNull()
		//c.AppendReply(nil, false)
		return
	}

	c.AppendFloat(Haversine(e1.Lat, e1.Lon, e2.Lat, e2.Lon))
}

func (GeoPosCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) < 2 {
		c.AppendError(errWrongArgs("geopos"))
		//	c.AppendReply(errWrongArgs("geopos"), false)
		return
	}

	key := args[0]

	obj, exists := Get(key)

	if !exists {
		c.AppendNull()
		//c.AppendReply(nil, false)
		return
	}

	if err := assertType(obj.TypeEncoding, OBJ_TYPE_GEO_SPATIAL); err != nil {
		c.AppendError(errWrongType())
		//c.AppendReply(errWrongType(), false)
		return
	}

	g := obj.Value.(*GeoSpatial)

	var res [][]string

	for _, member := range args[1:] {
		e, ok := g.index[member]
		if !ok {
			res = append(res, nil)
			continue
		}

		res = append(res, e.String())
	}

	c.AppendStringArrayArray(res)
}

func (GeoSearchCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 4 {
		c.AppendError(errWrongArgs("geosearch"))
		//c.AppendReply(errWrongArgs("geosearch"), false)
		return
	}

	key := args[0]
	obj, exists := Get(key)

	if !exists {
		c.AppendNull()
		//c.AppendReply(nil, false)
		return
	}

	if err := assertType(obj.TypeEncoding, OBJ_TYPE_GEO_SPATIAL); err != nil {
		c.AppendError(errWrongType())
		//	c.AppendReply(errWrongType(), false)
		return
	}

	g := obj.Value.(*GeoSpatial)

	lat, _ := strconv.ParseFloat(args[2], 64)
	lon, _ := strconv.ParseFloat(args[3], 64)

	c.AppendStrArray(g.Search(lat, lon))
}
