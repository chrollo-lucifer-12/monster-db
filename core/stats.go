package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis-server/resp"
)

type InfoCmd struct{}
type ClientCmd struct{}
type LatencyCmd struct{}
type SleepCmd struct{}

func (InfoCmd) Name() string    { return "INFO" }
func (ClientCmd) Name() string  { return "CLIENT" }
func (LatencyCmd) Name() string { return "LATENCY" }
func (SleepCmd) Name() string   { return "SLEEP" }

func (InfoCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	var info []byte
	buf := bytes.NewBuffer(info)
	buf.WriteString("# Keyspace\r\n")

	for i := range KeyspaceStat {
		buf.WriteString(fmt.Sprintf("db%d:keys=%d,expires=0,avg_ttl=0\r\n", i, KeyspaceStat[i]["keys"]))
	}

	return resp.Encode(buf.String(), false)
}

func (ClientCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	return RESP_OK
}

func (LatencyCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
	return resp.Encode([]string{}, false)
}

func (SleepCmd) Execute(ctx context.Context, c ClientCommander, args []string) []byte {
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

var KeyspaceStat [4]map[string]int

func UpdateDBStat(num int, metric string, value int) {
	KeyspaceStat[num][metric] = value
}
