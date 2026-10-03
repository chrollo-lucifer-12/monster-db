package core

var (
	RESP_NIL       = []byte("$-1\r\n")
	RESP_OK        = []byte("+OK\r\n")
	RESP_ZERO      = []byte(":0\r\n")
	RESP_ONE       = []byte(":1\r\n")
	RESP_MINUS_ONE = []byte(":-1\r\n")
	RESP_MINUS_TWO = []byte(":-2\r\n")

	errMsgWrongType  = "WRONGTYPE Operation against a key holding the wrong kind of value"
	errMsgInvalidInt = "ERR value is not an integer or out of range"
)

func errWrongArgs(cmd string) string {
	return "ERR wrong number of arguments for '" + cmd + "' command"
}

func errWrongType() string {
	return errMsgWrongType
}

func errInvalidInt() string {
	return errMsgInvalidInt
}

// only used in AOF/writeCommand, keep returning []byte
// func encodeTokens(tokens []string) []byte {
// 	return resp.Encode(nil, tokens, false)
// }
