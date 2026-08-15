package server

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/redis-server/core"
	"github.com/redis-server/resp"
	"golang.org/x/sys/unix"
)

const (
	MULTI_MODE     uint8 = 1 << 0
	CLIENT_BLOCKED uint8 = 1 << 1
	CLIENT_SUB     uint8 = 1 << 2
	CLIENT_CAS     uint8 = 1 << 3
)

type Multistate struct {
	cmds    core.RedisCmds
	aborted bool
}

type Client struct {
	Fd         int
	QueryBuf   []byte
	ReplyBuf   []byte
	flag       uint8
	multistate Multistate
	key        []byte
	when       time.Time

	subscriptions map[string]struct{}
}

func (c *Client) SetFlag(flag uint8)      { c.flag |= flag }
func (c *Client) ClearFlag(flag uint8)    { c.flag &^= flag }
func (c *Client) HasFlag(flag uint8) bool { return c.flag&flag != 0 }

func (c *Client) AppendBytesReply(val []byte) {
	c.ReplyBuf = resp.EncodeStringBytes(c.ReplyBuf, val)
}

func (c *Client) AppendArrayLen(len int) {
	c.ReplyBuf = resp.EncodeArrayLen(c.ReplyBuf, len)
}

func (c *Client) AppendIntArray(v []int64) {
	c.ReplyBuf = resp.EncodeArrayLen(c.ReplyBuf, len(v))

	for _, item := range v {
		c.ReplyBuf = resp.EncodeInt(c.ReplyBuf, item)
	}
}

func (c *Client) AppendSimpleString(val string) {
	c.ReplyBuf = resp.EncodeSimpleString(c.ReplyBuf, val)
}

func (c *Client) AppendBulkString(val string) {
	c.ReplyBuf = append(c.ReplyBuf, '$')
	c.ReplyBuf = strconv.AppendInt(c.ReplyBuf, int64(len(val)), 10)
	c.ReplyBuf = append(c.ReplyBuf, '\r', '\n')
	c.ReplyBuf = append(c.ReplyBuf, val...)
	c.ReplyBuf = append(c.ReplyBuf, '\r', '\n')
}

func (c *Client) AppendStrArray(v []string) {
	c.ReplyBuf = resp.EncodeArrayLen(c.ReplyBuf, len(v))

	for _, item := range v {
		c.ReplyBuf = resp.EncodeString(c.ReplyBuf, item)
	}
}

func (c *Client) AppendStringArrayArray(v [][]string) {
	c.ReplyBuf = resp.EncodeArrayLen(c.ReplyBuf, len(v))

	for _, arr := range v {
		if arr == nil {
			c.ReplyBuf = append(c.ReplyBuf, '$', '-', '1', '\r', '\n')
			continue
		}

		c.AppendStrArray(arr)
	}
}

func (c *Client) AppendFloat(val float64) {
	c.ReplyBuf = strconv.AppendFloat(c.ReplyBuf, val, 'g', -1, 64)
}

func (c *Client) AppendNull() {
	c.ReplyBuf = append(c.ReplyBuf, "$-1\r\n"...)
}

func (c *Client) AppendError(err string) {
	c.ReplyBuf = append(c.ReplyBuf, '-')
	c.ReplyBuf = append(c.ReplyBuf, err...)
	c.ReplyBuf = append(c.ReplyBuf, '\r', '\n')
}

func (c *Client) AppendIntReply(val int64) {
	c.ReplyBuf = resp.EncodeInt(c.ReplyBuf, val)
}

func (c *Client) ResetMultiState() {
	c.multistate.cmds = nil
	c.multistate.aborted = false
}

func (c *Client) QueueCommand(cmd *core.RedisCmd) {
	c.multistate.cmds = append(c.multistate.cmds, cmd)
}

func (c *Client) MultiCommands() core.RedisCmds {
	return c.multistate.cmds
}

func (c *Client) AbortMulti() {
	c.multistate.aborted = true
}

func (c *Client) IsMultiAborted() bool {
	return c.multistate.aborted
}

func (c *Client) Key() []byte { return c.key }

func (c *Client) SetKey(k []byte) { c.key = k }

func (c *Client) SubscribedKeys() []string {
	keys := make([]string, 0, len(c.subscriptions))
	for k := range c.subscriptions {
		keys = append(keys, k)
	}
	return keys
}
func (c *Client) Subscribe(key string) (int, bool) {
	if c.subscriptions == nil {
		c.subscriptions = make(map[string]struct{})
	}

	if _, exists := c.subscriptions[key]; exists {
		return len(c.subscriptions), false
	}

	subscribers[key] = append(subscribers[key], c)

	c.subscriptions[key] = struct{}{}
	c.flag |= CLIENT_SUB

	return len(c.subscriptions), true
}

func (c *Client) Unsubscribe(key string) (int, bool) {
	if _, exists := c.subscriptions[key]; !exists {
		return len(c.subscriptions), false
	}

	delete(c.subscriptions, key)

	subscribers[key] = removeClient(subscribers[key], c)
	if len(subscribers[key]) == 0 {
		delete(subscribers, key)
	}

	if len(c.subscriptions) == 0 {
		c.flag &^= CLIENT_SUB
	}
	return len(c.subscriptions), true
}

func (c *Client) Publish(key, message string) int {
	subs := subscribers[key]

	count := 0
	for _, sub := range subs {
		sub.AppendStrArray([]string{"message", key, message})
		clientsPendingWrite[sub.Fd] = sub
		count++
	}
	return count
}

func (c *Client) WatchKeys(keys []string) {
	for _, key := range keys {
		watchedKeys[key] = append(watchedKeys[key], c)
	}
}

func (c *Client) UnwatchAllKeys() {
	for key, clients := range watchedKeys {
		watchedKeys[key] = removeClient(clients, c)
		if len(watchedKeys[key]) == 0 {
			delete(watchedKeys, key)
		}
	}
}

func (c *Client) BlockOn(key string, timeoutMs int) {
	waitingKeys[key] = append(waitingKeys[key], c)
	log.Printf(
		"SETTING BLOCKED: client=%p fd=%d flags_before=%08b",
		c, c.Fd, c.flag,
	)

	c.flag |= CLIENT_BLOCKED

	if timeoutMs > 0 {
		c.when = time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	} else {
		c.when = time.Time{}
	}
}

func freeClient(loop *EventLoop, client *Client) {
	loop.DeleteFileEvent(client.Fd, unix.EPOLLIN|unix.EPOLLOUT)
	unix.Close(client.Fd)
	con_clients--

}

func (c *Client) BlockClient(key string) {
	c.flag |= CLIENT_BLOCKED
	waitingKeys[key] = append(waitingKeys[key], c)
}

func isSlowClient(client *Client) bool {
	bufSize := len(client.ReplyBuf)

	if bufSize >= PubSubHardLimit {
		return true
	}

	return false
}

func cleanupClient(client *Client, loop *EventLoop) {
	freeClient(loop, client)

	delete(clientsPendingWrite, client.Fd)

	for key := range client.subscriptions {
		subscribers[key] = removeClient(subscribers[key], client)
		if len(subscribers[key]) == 0 {
			delete(subscribers, key)
		}
	}

	client.ReplyBuf = nil
	client.subscriptions = nil
}

func processKeys(loop *EventLoop) {
	for key := range readyKeys {
		waitingClients := waitingKeys[key]

		if len(waitingClients) == 0 {
			delete(readyKeys, key)
			continue
		}

		client := waitingClients[0]
		ctx := core.WithClient(context.Background(), client)

		cmdImpl, _ := core.Lookup("LPOP")

		cmdImpl.Execute(ctx, client, []string{key})

		waitingKeys[key] = waitingClients[1:]
		client.flag &= ^CLIENT_BLOCKED
		client.when = time.Time{}

		clientsPendingWrite[client.Fd] = client

		delete(readyKeys, key)
	}

}

func HandleBlockedClients(loop *EventLoop, id int64, clientData any) int {

	now := time.Now()

	for key, clients := range waitingKeys {
		var activeClients []*Client

		for _, client := range clients {

			if !client.when.IsZero() && now.After(client.when) {
				client.flag &= ^CLIENT_BLOCKED
				client.when = time.Time{}

				client.ReplyBuf = append(client.ReplyBuf, []byte("$-1\r\n")...)

				clientsPendingWrite[client.Fd] = client
			} else {
				activeClients = append(activeClients, client)
			}

		}

		if len(activeClients) == 0 {
			delete(waitingKeys, key)
		} else {
			waitingKeys[key] = activeClients
		}
	}

	return 100
}

func NewClient(fd int) *Client {

	qBuf := make([]byte, 4096)

	rBuf := make([]byte, 4096)

	return &Client{
		Fd:            fd,
		QueryBuf:      qBuf[:0],
		ReplyBuf:      rBuf[:0],
		subscriptions: make(map[string]struct{}),
	}
}
