package server

import (
	"log"
	"time"

	"github.com/redis-server/alloc"
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

func (c *Client) AppendReply(b []byte) {
	c.ReplyBuf = append(c.ReplyBuf, b...)
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
		sub.ReplyBuf = append(sub.ReplyBuf, resp.Encode([]string{"message", key, message}, false)...)
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

	alloc.Free(client.QueryBuf)
	alloc.Free(client.ReplyBuf)
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

func NewClient(fd int) *Client {

	qBuf, err := alloc.Alloc(4096)
	if err != nil {
		log.Println("OOM")
		return nil
	}

	rBuf, err := alloc.Alloc(4096)
	if err != nil {
		log.Println("OOM")
		alloc.Free(qBuf)
		return nil
	}

	return &Client{
		Fd:            fd,
		QueryBuf:      qBuf[:0],
		ReplyBuf:      rBuf[:0],
		subscriptions: make(map[string]struct{}),
	}
}
