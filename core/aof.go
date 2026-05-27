package core

import (
	"fmt"
	"io"
	"log"
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
	fp, err := os.OpenFile(config.AOFFILE, os.O_CREATE|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		fmt.Println("error", err)
		return
	}

	defer fp.Close()

	log.Println("rewriting AOF file at ", config.AOFFILE)

	for k, obj := range store {
		dumpkey(fp, k, obj)
	}

	fp.Sync()

	log.Println("AOF file rewrite complete")
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

	fmt.Println(decoded...)
}
