package core

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/redis-server/config"
	"github.com/redis-server/resp"
)

func dumpkey(fp *os.File, k string, obj *Obj) {
	cmd := fmt.Sprintf("SET %s %s", k, obj.Value)
	tokens := strings.Split(cmd, " ")
	fp.Write(resp.Encode(tokens, false))
}

func DumpAllAOF() {
	fp, err := os.OpenFile(config.AOFFILE, os.O_CREATE|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		fmt.Println("error", err)
		return
	}

	log.Println("rewriting AOF file at ", config.AOFFILE)

	for k, obj := range store {
		dumpkey(fp, k, obj)
	}

	log.Println("AOF file rewrite complete")
}
