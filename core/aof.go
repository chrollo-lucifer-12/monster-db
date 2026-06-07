package core

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/redis-server/config"
	"github.com/redis-server/resp"
)

func writeCommand(fp *os.File, tokens []string) {
	fp.Write(resp.Encode(tokens, false))
}

func dumpkey(fp *os.File, k string, obj *Obj) {
	writeCommand(fp, []string{
		"SET",
		k,
		obj.Value.(string),
	})

	exp, isExpirySet := getExpiry(obj)

	if isExpirySet {
		writeCommand(fp, []string{
			"EXPIRE",
			k,
			strconv.FormatInt(int64(exp), 10),
		})
	}
}

func DumpAllAOF() {
	tempFile := config.AOFFILE + ".tmp"
	fp, _ := os.OpenFile(tempFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)

	for k, obj := range store {
		dumpkey(fp, k, obj)
	}

	fp.Sync()
	fp.Close()

	os.Rename(tempFile, config.AOFFILE)
}

func Restoreaof() {
	fp, err := os.OpenFile(config.AOFFILE, os.O_RDONLY, 0644)
	if err != nil {
		fmt.Println("error", err)
		return
	}

	defer fp.Close()

	data, err := io.ReadAll(fp)
	if err != nil {
		fmt.Println("error", err)
		return
	}

	decoded, _, err := resp.Decode(data)
	if err != nil {
		fmt.Println("decode error:", err)
		return
	}

	for _, raw := range decoded {
		arr, ok := raw.([]interface{})
		if !ok || len(arr) == 0 {
			continue
		}

		cmdName, ok := arr[0].(string)
		if !ok {
			continue
		}

		args := make([]string, 0, len(arr)-1)

		for _, v := range arr[1:] {
			if s, ok := v.(string); ok {
				args = append(args, s)
			}
		}

		cmd := RedisCmd{
			Cmd:  cmdName,
			Args: args,
		}

		Eval(&cmd)
	}

	fmt.Println("AOF replay done")
}
