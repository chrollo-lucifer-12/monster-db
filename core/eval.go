package core

import (
	"errors"

	"github.com/redis-server/resp"
)

var RESP_NIL []byte = []byte("$-1\r\n")
var RESP_OK []byte = []byte("+OK\r\n")
var RESP_ZERO []byte = []byte(":0\r\n")
var RESP_ONE []byte = []byte(":1\r\n")
var RESP_MINUS_ONE []byte = []byte(":-1\r\n")
var RESP_MINUS_TWO []byte = []byte(":-2\r\n")
var RESP_WRONG_TYPE []byte = []byte("+WRONG_TYPE\r\n")

func errWrongArgs(cmd string) []byte {
	return resp.Encode(errors.New(
		"ERR wrong number of arguments for '"+cmd+"' command",
	), false)
}

func errWrongType() []byte {
	return resp.Encode(errors.New(
		"WRONGTYPE Operation against a key holding the wrong kind of value",
	), false)
}

func errInvalidInt() []byte {
	return resp.Encode(errors.New(
		"ERR value is not an integer or out of range",
	), false)
}
