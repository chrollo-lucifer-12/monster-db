package core

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/redis-server/config"
	"github.com/redis-server/resp"
)

func EncodeStore() []byte {

	var buf []byte

	for k, obj := range store {
		cmd := resp.Encode([]string{
			"SET",
			k,
			string(obj.Value.([]byte)),
		}, false)

		buf = append(buf, cmd...)
	}

	return buf
}

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
	fp, err := os.OpenFile(config.RDBFILE, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
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

	r1, _, err1 := syscall.RawSyscall(syscall.SYS_FORK, 0, 0, 0)

	if err1 != 0 {
		log.Println("Fork failed:", err1)
		return []byte("-ERR background save failed\r\n")
	}

	if r1 == 0 {
		SaveRDB()
		os.Exit(0)
	}

	log.Printf("Background save started in child process (PID: %d)\n", r1)
	return RESP_OK

}
