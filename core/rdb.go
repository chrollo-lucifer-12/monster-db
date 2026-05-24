package core

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/redis-server/config"
)

func LoadRDB() {
	fp, err := os.OpenFile(config.RDBFILE, os.O_RDONLY, 0644)
	if err != nil {
		fmt.Println("error", err)
		return
	}

	defer fp.Close()

	for {
		var keyLen int32

		err := binary.Read(fp, binary.LittleEndian, &keyLen)
		if err != nil {
			break
		}

		key := make([]byte, keyLen)
		fp.Read(key)

		typeBuf := make([]byte, 1)
		fp.Read(typeBuf)

		var expiresAt uint32
		binary.Read(fp, binary.LittleEndian, &expiresAt)

		var valLen int32
		binary.Read(fp, binary.LittleEndian, &valLen)

		value := make([]byte, valLen)
		fp.Read(value)

		store[string(key)] = &Obj{
			TypeEncoding:   typeBuf[0],
			LastAccessedAt: expiresAt,
			Value:          value,
		}
	}
}

func SaveRDB() {
	fp, err := os.OpenFile(config.RDBFILE, os.O_CREATE|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		fmt.Println("error", err)
		return
	}

	defer fp.Close()

	for k, obj := range store {
		keyBytes := []byte(k)

		if hasExpired(obj) {
			continue
		}

		binary.Write(fp, binary.LittleEndian, int32(len(keyBytes)))
		fp.Write(keyBytes)

		fp.Write([]byte{obj.TypeEncoding})

		binary.Write(fp, binary.LittleEndian, obj.LastAccessedAt)

		binary.Write(fp, binary.LittleEndian, int32(len(obj.Value.([]byte))))
		fp.Write(obj.Value.([]byte))
	}

	fp.Sync()
}

func TriggerRDB() []byte {

	cmd := exec.Command("go", "run", "main.go", "--rdb-dump")

	err := cmd.Start()
	if err != nil {
		return RESP_MINUS_ONE
	}

	log.Println("dump successful")

	return RESP_OK
}
