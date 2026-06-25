package core

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

var (
	errWrongArgsSleep  = "ERR wrong number of arguments for 'sleep' command"
	errInvalidDuration = "ERR value is not an integer or out of range"
)

type InfoCmd struct{}
type ClientCmd struct{}
type LatencyCmd struct{}
type SleepCmd struct{}

func (InfoCmd) Name() string    { return "INFO" }
func (ClientCmd) Name() string  { return "CLIENT" }
func (LatencyCmd) Name() string { return "LATENCY" }
func (SleepCmd) Name() string   { return "SLEEP" }

func (InfoCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	var buf []byte
	buf = append(buf, "# Keyspace\r\n"...)

	for i := range KeyspaceStat {
		line := fmt.Sprintf("db%d:keys=%d,expires=0,avg_ttl=0\r\n", i, KeyspaceStat[i]["keys"])
		buf = append(buf, line...)
	}

	c.AppendReply(string(buf), false)
}

func (ClientCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	c.AppendReply(RESP_OK, true)
}

func (LatencyCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	c.AppendReply([]string{}, false)
}

func (SleepCmd) Execute(ctx context.Context, c ClientCommander, args []string) {
	if len(args) != 1 {
		c.AppendReply(errWrongArgsSleep, false)
		return
	}

	durationSec, err := strconv.ParseInt(args[0], 10, 64)

	if err != nil {
		c.AppendReply(errInvalidDuration, false)
		return
	}

	time.Sleep(time.Duration(durationSec) * time.Second)

	c.AppendReply(RESP_OK, true)
}

var KeyspaceStat [4]map[string]int

func UpdateDBStat(num int, metric string, value int) {
	KeyspaceStat[num][metric] = value
}
