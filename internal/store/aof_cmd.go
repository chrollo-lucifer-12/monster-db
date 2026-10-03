package core

import (
	"context"
	"log/slog"
	"os"
	"syscall"

	"github.com/redis-server/internal/runtime"
)

type AOFCmd struct{}

func (AOFCmd) Name() string { return "BGREWRITEAOF" }

func (AOFCmd) Execute(ctx context.Context, c runtime.ClientCommander, args []string) {
	r1, _, err1 := syscall.RawSyscall(syscall.SYS_FORK, 0, 0, 0)

	if err1 != 0 {
		slog.Error("Fork failed", "err", err1)
		c.AppendError("fork failes")
		return
	}

	if r1 == 0 {
		DumpAllAOF()
		os.Exit(0)
	}

	slog.Info("Background save started in child process", "pid", r1)
	c.AppendSimpleString("RESP_OK")
}
