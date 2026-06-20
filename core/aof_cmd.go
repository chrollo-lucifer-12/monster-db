package core

import (
	"context"
	"log"
	"os"
	"syscall"
)

type AOFCmd struct{}

func (AOFCmd) Name() string { return "BGREWRITEAOF" }

func (AOFCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
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
