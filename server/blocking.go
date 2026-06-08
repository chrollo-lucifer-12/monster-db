package server

import (
	"time"

	"github.com/redis-server/core"
)

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

func processKeys(loop *EventLoop) {
	for key := range readyKeys {
		waitingClients := waitingKeys[key]

		if len(waitingClients) == 0 {
			delete(readyKeys, key)
			continue
		}

		res := core.Eval(&core.RedisCmd{
			Cmd:  "LPOP",
			Args: []string{key},
		})

		client := waitingClients[0]
		waitingKeys[key] = waitingClients[1:]
		client.flag &= ^CLIENT_BLOCKED
		client.when = time.Time{}

		client.ReplyBuf = append(client.ReplyBuf, res...)
		clientsPendingWrite[client.Fd] = client

		delete(readyKeys, key)
	}

}
