package server

import "github.com/redis-server/resp"

func HandleSubscribe(client *Client, args []string) {
	for _, key := range args {

		if _, exists := client.subscriptions[key]; exists {
			continue
		}

		subscribers[key] = append(subscribers[key], client)

		client.subscriptions[key] = struct{}{}
		client.flag |= CLIENT_SUB

		client.ReplyBuf = append(client.ReplyBuf, resp.Encode([]any{"subscribe", key, len(client.subscriptions)}, false)...)
	}
}

func HandleUnsubscribe(client *Client, args []string) {
	var targets []string
	if len(args) > 0 {
		targets = args
	} else {
		for key := range client.subscriptions {
			targets = append(targets, key)
		}
	}

	for _, key := range targets {
		if _, exists := client.subscriptions[key]; !exists {
			continue
		}

		delete(client.subscriptions, key)
		subscribers[key] = removeClient(subscribers[key], client)

		if len(subscribers[key]) == 0 {
			delete(subscribers, key)
		}

		client.ReplyBuf = append(client.ReplyBuf, resp.Encode([]any{"unsubscribe", key, len(client.subscriptions)}, false)...)
	}

	if len(client.subscriptions) == 0 {
		client.flag &= ^CLIENT_SUB
	}
}

func HandlePublish(client *Client, args []string) {
	key := args[0]
	message := args[1]

	c := 0

	for _, sub_client := range subscribers[key] {
		sub_client.ReplyBuf = append(sub_client.ReplyBuf, resp.Encode([]string{"message", key, message}, false)...)
		clientsPendingWrite[sub_client.Fd] = sub_client
		c++
	}

	client.ReplyBuf = append(client.ReplyBuf, resp.Encode(c, false)...)
}
