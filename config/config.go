package config

var Host string
var Port int
var KeyLimit int
var AOFFILE string = "./appendonly.aof"

var KeysLimit int = 100
var EvictionRatio float64 = 0.40
var EvictionStrategy string = "allkeys-random"
