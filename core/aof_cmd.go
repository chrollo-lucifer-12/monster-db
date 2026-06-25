package core

import (
	"context"
	"log"
	"os"
	"syscall"
)

type AOFCmd struct{}

func (AOFCmd) Name() string { return "BGREWRITEAOF" }

func (AOFCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	r1, _, err1 := syscall.RawSyscall(syscall.SYS_FORK, 0, 0, 0)

	if err1 != 0 {
		log.Println("Fork failed:", err1)
		c.AppendReply("Fork failed", false)
		return
	}

	if r1 == 0 {
		DumpAllAOF()
		os.Exit(0)
	}

	log.Printf("Background save started in child process (PID: %d)\n", r1)
	c.AppendReply(RESP_OK, true)
}
