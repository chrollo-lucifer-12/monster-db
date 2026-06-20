package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"syscall"
	"time"

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

func evalPING(args []string) []byte {
	var b []byte

	if len(args) >= 2 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'ping' command"), false)
	}

	if len(args) == 0 {
		b = resp.Encode("PONG", true)
	} else {
		b = resp.Encode(args[0], false)
	}

	return b
}

func evalSET(args []string) []byte {
	if len(args) <= 1 {
		return resp.Encode(errors.New("(error) ERR wrong number of arguments for 'set' command"), false)
	}

	var key, value string
	var exDurationsMs int64 = -1

	key, value = args[0], args[1]
	oType, oEnc := deduceTypeEncoding(value)

	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "EX", "ex":
			i++
			if i == len(args) {
				return resp.Encode(errors.New("(error) ERR syntax error"), false)
			}

			exDurationSec, err := strconv.ParseInt(args[3], 10, 64)
			if err != nil {
				return resp.Encode(err, false)
			}

			exDurationsMs = exDurationSec * 1000

		default:
			return resp.Encode(errors.New("(error) ERR syntax error"), false)
		}
	}
	Put(key, NewObj(value, exDurationsMs, oType, oEnc))
	return RESP_OK
}

func evalGET(args []string) []byte {
	if len(args) != 1 {
		return resp.Encode(errors.New("(error) ERR wrong number of arguments for 'get' command"), false)
	}

	var key string = args[0]

	obj := Get(key)

	if obj == nil {
		return RESP_NIL
	}

	if hasExpired(obj) {
		return RESP_NIL
	}

	return resp.Encode(obj.Value, false)
}

func evalTTL(args []string) []byte {
	if len(args) != 1 {
		return resp.Encode(errors.New("(error) ERR wrong number of arguments for 'get' command"), false)
	}

	var key string = args[0]

	obj := Get(key)

	if obj == nil {
		return RESP_MINUS_TWO
	}

	exp, isExpirySet := getExpiry(obj)

	if !isExpirySet {
		return RESP_MINUS_ONE
	}

	if uint64(time.Now().UnixMilli()) > exp {
		return RESP_MINUS_TWO
	}

	durationMs := exp - uint64(time.Now().UnixMilli())

	return resp.Encode((durationMs / 1000), false)
}

func evalDEL(args []string) []byte {
	var countDeleted int = 0

	for _, key := range args {
		if ok := Del(key); ok {
			countDeleted++
		}
	}

	return resp.Encode(countDeleted, false)

}

func evalEXPIRE(args []string) []byte {
	if len(args) <= 1 {
		return resp.Encode(errors.New("(error) ERR wrong number of arguments for 'espire' command"), false)
	}

	var key string = args[0]
	exDurationSec, err := strconv.ParseInt(args[1], 10, 64)

	if err != nil {
		return resp.Encode(errors.New("(error) ERR value is not an integer or out of range"), false)
	}

	obj := Get(key)

	if obj == nil {
		return RESP_ZERO
	}

	setExpiry(obj, exDurationSec*1000)

	return RESP_ONE
}

func evalBGREWRITE(args []string) []byte {

	r1, _, err1 := syscall.RawSyscall(syscall.SYS_FORK, 0, 0, 0)

	if err1 != 0 {
		log.Println("Fork failed:", err1)
		return []byte("-ERR background save failed\r\n")
	}

	if r1 == 0 {
		DumpAllAOF()
		os.Exit(0)
	}

	log.Printf("Background save started in child process (PID: %d)\n", r1)
	return RESP_OK
}

func evalINCR(args []string) []byte {
	if len(args) != 1 {
		return resp.Encode(errors.New("(error) ERR wrong number of arguments for 'incr' command"), false)
	}

	var key string = args[0]
	obj := Get(key)

	if obj == nil {
		obj = NewObj("0", -1, OBJ_TYPE_STRING, OBJ_ENCODING_INT)
		Put(key, obj)
	}

	if err := assertType(obj.TypeEncoding, OBJ_TYPE_STRING); err != nil {
		return resp.Encode(err, false)
	}

	if err := assertEncoding(obj.TypeEncoding, OBJ_ENCODING_INT); err != nil {
		return resp.Encode(err, false)
	}

	i, _ := strconv.ParseInt(obj.Value.(string), 10, 64)
	i++
	obj.Value = strconv.FormatInt(i, 10)

	return resp.Encode(i, false)
}

func evalINFO(args []string) []byte {
	var info []byte
	buf := bytes.NewBuffer(info)
	buf.WriteString("# Keyspace\r\n")

	for i := range KeyspaceStat {
		buf.WriteString(fmt.Sprintf("db%d:keys=%d,expires=0,avg_ttl=0\r\n", i, KeyspaceStat[i]["keys"]))
	}

	return resp.Encode(buf.String(), false)
}

func evalCLIENT(args []string) []byte {
	return RESP_OK
}

func evalLATENCY(args []string) []byte {
	return resp.Encode([]string{}, false)
}

func evalSLEEP(args []string) []byte {
	if len(args) != 1 {
		return resp.Encode(errors.New("(error) ERR wrong number of arguments for 'sleep' command"), false)
	}

	durationSec, err := strconv.ParseInt(args[0], 10, 64)

	if err != nil {
		return resp.Encode(errors.New("ERR value is not an integer or out of range"), false)
	}

	time.Sleep(time.Duration(durationSec) * time.Second)

	return RESP_OK
}

func evalLLEN(args []string) []byte {
	if len(args) != 1 {
		return resp.Encode(errors.New("(error) ERR wrong number of arguments for 'llen' command"), false)
	}

	var key string = args[0]
	obj := Get(key)

	if obj == nil {
		return RESP_ZERO
	}

	if err := assertType(obj.TypeEncoding, OBJ_TYPE_LIST); err != nil {
		return RESP_ZERO
	}

	ql := obj.Value.(*Quicklist)

	return resp.Encode(ql.len, false)
}

func evalLPUSH(args []string) []byte {
	if len(args) < 2 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'lpush' command"), false)
	}

	var key string = args[0]
	obj := Get(key)

	var ql *Quicklist

	if obj == nil {
		ql = NewQuicklist()
		obj = NewObj(ql, -1, OBJ_TYPE_LIST, OBJ_ENCODING_LISTPACK)
		Put(key, obj)
	} else {
		if err := assertType(obj.TypeEncoding, OBJ_TYPE_LIST); err != nil {
			return resp.Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
		}

		ql = obj.Value.(*Quicklist)
	}

	oldLen := ql.len

	for i := 1; i < len(args); i++ {
		val, err := strconv.Atoi(args[i])
		if err != nil {
			ql.addToHead(args[i])
		} else {
			ql.addToHead(val)
		}
	}

	if oldLen == 0 {
		MarkReady(key)
	}

	return resp.Encode(ql.len, false)
}

func evalRPUSH(args []string) []byte {
	if len(args) < 2 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'lpush' command"), false)
	}

	var key string = args[0]
	obj := Get(key)

	var ql *Quicklist

	if obj == nil {
		ql = NewQuicklist()
		obj = NewObj(ql, -1, OBJ_TYPE_LIST, OBJ_ENCODING_LISTPACK)
		Put(key, obj)
	} else {
		if err := assertType(obj.TypeEncoding, OBJ_TYPE_LIST); err != nil {
			return resp.Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
		}

		ql = obj.Value.(*Quicklist)
	}

	oldLen := ql.len

	for i := 1; i < len(args); i++ {
		val, err := strconv.Atoi(args[i])
		if err != nil {
			ql.addToTail(args[i])
		} else {
			ql.addToTail(val)
		}
	}

	if oldLen == 0 {
		MarkReady(key)
	}

	return resp.Encode(ql.len, false)
}

func evalLRANGE(args []string) []byte {
	if len(args) != 3 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'lrange' command"), false)
	}

	var key string = args[0]
	start, _ := strconv.Atoi(args[1])
	stop, _ := strconv.Atoi(args[2])

	obj := Get(key)

	if obj == nil {
		return RESP_NIL
	}

	ql := obj.Value.(*Quicklist)

	res := ql.GetElements(start, stop)

	return resp.Encode(res, false)
}

func evalLPOP(args []string) []byte {
	if len(args) < 1 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'lpop' command"), false)
	}

	var key string = args[0]
	count := 1

	if len(args) == 2 {
		count, _ = strconv.Atoi(args[1])
	}

	obj := Get(key)

	if obj == nil {
		return RESP_NIL
	}

	ql := obj.Value.(*Quicklist)

	res := ql.RemoveElements(count)

	return resp.Encode(res, false)
}

func evalBFRESERVE(args []string) []byte {
	if len(args) < 3 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'bfreserve' command"), false)
	}

	key := args[0]
	errorRate, _ := strconv.ParseFloat(args[1], 32)
	capacity, _ := strconv.Atoi(args[2])

	bl := NewBloomFilter(capacity, errorRate)

	obj := NewObj(bl, -1, uint8(OBJ_TYPE_BLOOM_FILTERS), OBJ_ENCODING_BOOL_ARR)

	Put(key, obj)

	return RESP_OK
}

func evalBFADD(args []string) []byte {
	if len(args) != 2 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'bfadd' command"), false)
	}

	key := args[0]
	pat := args[1]

	obj := Get(key)

	if obj == nil {
		return RESP_NIL
	}

	bl := obj.Value.(*BloomFilter)

	bl.Add(pat)

	return RESP_ONE
}

func evalBFEXISTS(args []string) []byte {
	if len(args) != 2 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'bfadd' command"), false)
	}

	key := args[0]
	pat := args[1]

	obj := Get(key)

	if obj == nil {
		return RESP_NIL
	}

	if err := assertType(obj.TypeEncoding, uint8(OBJ_TYPE_BLOOM_FILTERS)); err != nil {
		return resp.Encode(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"), false)
	}

	bl := obj.Value.(*BloomFilter)

	if bl.Exists(pat) {
		return RESP_ONE
	}

	return RESP_ZERO
}

func evalSADD(args []string) []byte {
	if len(args) < 2 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'sadd' command"), false)
	}

	key := args[0]
	obj := Get(key)

	var is *Intset

	if obj == nil {
		is = NewIntset()
		Put(key, NewObj(is, -1, uint8(OBJ_TYPE_SET), OBJ_ENCODING_INSET))
	} else {
		is = obj.Value.(*Intset)
	}

	c := 0

	for _, v := range args[1:] {
		n, err := strconv.ParseInt(v, 10, 16)
		if err == nil {
			inserted := is.set(int16(n))
			if inserted {
				c++
			}
		}
	}

	return resp.Encode(c, false)
}

func evalSCARD(args []string) []byte {
	if len(args) != 1 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'scard' command"), false)
	}

	key := args[0]

	obj := Get(key)

	if err := assertType(obj.TypeEncoding, uint8(OBJ_TYPE_SET)); err != nil {
		return RESP_NIL
	}

	is := obj.Value.(*Intset)

	return resp.Encode(is.length, false)
}

func evalSISMEMBER(args []string) []byte {
	if len(args) != 2 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'sismember' command"), false)
	}

	key := args[0]

	obj := Get(key)

	if err := assertType(obj.TypeEncoding, uint8(OBJ_TYPE_SET)); err != nil {
		return RESP_NIL
	}

	value, _ := strconv.ParseInt(args[1], 10, 16)

	is := obj.Value.(*Intset)

	idx := is.search(int16(value))

	if idx != -1 && is.get(idx) == int16(value) {
		return RESP_ONE
	}

	return RESP_ZERO
}

func evalSMEMBERS(args []string) []byte {
	if len(args) != 1 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'smembers' command"), false)
	}

	key := args[0]

	obj := Get(key)

	if err := assertType(obj.TypeEncoding, uint8(OBJ_TYPE_SET)); err != nil {
		return RESP_NIL
	}

	elements := []any{}
	is := obj.Value.(*Intset)

	for i := 0; i < int(is.length); i++ {
		elements = append(elements, is.get(i))
	}

	return resp.Encode(elements, false)
}

func evalSREM(args []string) []byte {
	if len(args) != 2 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'srem' command"), false)
	}

	key := args[0]

	obj := Get(key)

	if err := assertType(obj.TypeEncoding, uint8(OBJ_TYPE_SET)); err != nil {
		return RESP_NIL
	}

	value, _ := strconv.ParseInt(args[1], 10, 16)

	is := obj.Value.(*Intset)

	return resp.Encode(is.del(int16(value)), false)
}

func evalZADD(args []string) []byte {
	if len(args) != 3 {
		return errWrongArgs("zadd")
	}

	key := args[0]
	score, err := strconv.ParseInt(args[1], 10, 64)

	if err != nil {
		return errInvalidInt()
	}

	member := args[2]

	obj := Get(key)

	var z *Zset

	if obj == nil {
		z = NewZset()
		Put(key, NewObj(z, -1, OBJ_TYPE_ZSET, OBJ_ENCODING_SKIPLIST))
	} else {
		z = obj.Value.(*Zset)
	}

	z.Add(member, int(score))

	return RESP_ONE
}

func evalZREM(args []string) []byte {
	if len(args) < 2 {
		return errWrongArgs("zrem")
	}

	key := args[0]
	obj := Get(key)

	if obj == nil {
		return RESP_NIL
	}

	if err := assertType(obj.TypeEncoding, OBJ_TYPE_ZSET); err != nil {
		return errWrongType()
	}

	zs := obj.Value.(*Zset)

	c := 0

	for _, member := range args[1:] {
		c += zs.Delete(member)
	}

	return resp.Encode(c, false)
}

func evalZSCORE(args []string) []byte {
	if len(args) != 2 {
		return errWrongArgs("zscore")
	}

	key := args[0]
	member := args[1]
	obj := Get(key)

	if obj == nil {
		return RESP_NIL
	}

	if err := assertType(obj.TypeEncoding, OBJ_TYPE_ZSET); err != nil {
		return errWrongType()
	}

	zs := obj.Value.(*Zset)

	score, exists := zs.Search(member)

	if !exists {
		return RESP_NIL
	}

	return resp.Encode(score, false)
}

func evalZRANGE(args []string) []byte {
	if len(args) != 3 {
		return resp.Encode(errors.New("ERR wrong number of arguments for 'zscore' command"), false)
	}

	key := args[0]
	start, _ := strconv.ParseInt(args[1], 10, 64)
	stop, _ := strconv.ParseInt(args[2], 10, 64)

	obj := Get(key)

	if obj == nil {
		return RESP_NIL
	}

	if err := assertType(obj.TypeEncoding, OBJ_TYPE_ZSET); err != nil {
		return RESP_NIL
	}

	zs := obj.Value.(*Zset)

	members := zs.Range(int(start), int(stop))

	return resp.Encode(members, false)
}

func Eval(cmd *RedisCmd) []byte {

	switch cmd.Cmd {
	case "SET":
		return evalSET(cmd.Args)
	case "GET":
		return evalGET(cmd.Args)
	case "TTL":
		return evalTTL(cmd.Args)
	case "DEL":
		return evalDEL(cmd.Args)
	case "EXPIRE":
		return evalEXPIRE(cmd.Args)
	case "BGREWRITEAOF":
		return evalBGREWRITE(cmd.Args)
	case "INCR":
		return evalINCR(cmd.Args)
	case "INFO":
		return evalINFO(cmd.Args)
	case "CLIENT":
		return evalCLIENT(cmd.Args)
	case "LATENCY":
		return evalLATENCY(cmd.Args)
	case "SLEEP":
		return evalSLEEP(cmd.Args)
	case "RPUSH":
		return evalRPUSH(cmd.Args)
	case "LPUSH":
		return evalLPUSH(cmd.Args)
	case "LLEN":
		return evalLLEN(cmd.Args)
	case "LRANGE":
		return evalLRANGE(cmd.Args)
	case "LPOP":
		return evalLPOP(cmd.Args)
	case "BF.ADD":
		return evalBFADD(cmd.Args)
	case "BF.EXISTS":
		return evalBFEXISTS(cmd.Args)
	case "BF.RESERVE":
		return evalBFRESERVE(cmd.Args)
	case "SADD":
		return evalSADD(cmd.Args)
	case "SCARD":
		return evalSCARD(cmd.Args)
	case "SISMEMBER":
		return evalSISMEMBER(cmd.Args)
	case "SMEMBERS":
		return evalSMEMBERS(cmd.Args)
	case "SREM":
		return evalSREM(cmd.Args)
	case "ZADD":
		return evalZADD(cmd.Args)
	case "ZREM":
		return evalZREM(cmd.Args)
	case "ZSCORE":
		return evalZSCORE(cmd.Args)
	case "ZRANGE":
		return evalZRANGE(cmd.Args)
	default:
		return evalPING(cmd.Args)
	}

}

func EvalCtx(ctx context.Context, redisCmd *RedisCmd) []byte {
	cmd, ok := registry[redisCmd.Cmd]
	if !ok {
		cmd = PingCmd{}
	}

	client, ok := ClientFromContext(ctx)
	if !ok {
		return resp.Encode(errors.New("ERR no client in context"), false)
	}

	return cmd.Execute(ctx, client, redisCmd.Args)
}
