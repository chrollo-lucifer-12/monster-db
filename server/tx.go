package server

import (
	"errors"

	"github.com/redis-server/core"
	"github.com/redis-server/resp"
)

func HandleMulti(client *Client, args []string) {
	client.flag |= MULTI_MODE
	client.multistate.cmds = nil
	client.ReplyBuf = append(client.ReplyBuf, []byte("+OK\r\n")...)
}

func HandleWatch(client *Client, args []string) error {
	if client.flag&MULTI_MODE != 0 {
		client.ReplyBuf = append(client.ReplyBuf, resp.Encode("WATCH inside MULTI not allowed", false)...)
		return errors.New("")
	}

	for _, key := range args {
		watchedKeys[key] = append(watchedKeys[key], client)
	}

	client.ReplyBuf = append(client.ReplyBuf, core.RESP_OK...)

	return nil
}

func HandleExec(client *Client, args []string) error {
	if client.flag&MULTI_MODE == 0 {
		client.ReplyBuf = append(client.ReplyBuf,
			[]byte("-ERR EXEC without MULTI\r\n")...)
		return errors.New("")
	}

	if client.flag&CLIENT_CAS != 0 {
		UnwatchAllKeys(client)
		client.multistate.cmds = nil
		client.flag &= ^MULTI_MODE

		client.ReplyBuf = append(client.ReplyBuf, core.RESP_NIL...)
		return errors.New("")
	}

	if len(client.multistate.cmds) == 0 {
		client.flag &= ^MULTI_MODE
		client.ReplyBuf = append(client.ReplyBuf, []byte("*0\r\n")...)
		return errors.New("")
	}

	results := make([][]byte, 0, len(client.multistate.cmds))

	for _, qcmd := range client.multistate.cmds {
		results = append(results, core.Eval(qcmd))
	}

	client.multistate.cmds = nil
	client.flag &= ^MULTI_MODE

	UnwatchAllKeys(client)

	client.ReplyBuf = append(client.ReplyBuf, resp.EncodeExecArray(results)...)

	return nil
}

func HandleDiscard(client *Client, args []string) error {
	if client.flag&MULTI_MODE == 0 {
		client.ReplyBuf = append(client.ReplyBuf,
			[]byte("-ERR DISCARD without MULTI\r\n")...)
		return errors.New("")
	}

	client.multistate.cmds = nil
	client.flag &= ^MULTI_MODE

	UnwatchAllKeys(client)

	client.ReplyBuf = append(client.ReplyBuf, []byte("+OK\r\n")...)

	return nil
}
