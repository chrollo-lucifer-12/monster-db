package server

import (
	"log"

	"golang.org/x/sys/unix"
)

func ReadQueryFromClient(loop *EventLoop, fd int, clientData interface{}) {
	client := clientData.(*Client)

	if (client.flag & CLIENT_BLOCKED) != 0 {
		return
	}

	for {

		if len(client.QueryBuf) == cap(client.QueryBuf) {
			newCap := cap(client.QueryBuf) * 2
			if newCap == 0 {
				newCap = 4096
			}

			newBuf := make([]byte, newCap)

			newBuf = newBuf[:len(client.QueryBuf)]
			copy(newBuf, client.QueryBuf)

			client.QueryBuf = newBuf[:len(client.QueryBuf)]
		}

		freeSpace := client.QueryBuf[len(client.QueryBuf):cap(client.QueryBuf)]

		n, err := unix.Read(fd, freeSpace)
		if err != nil {

			if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
				break
			}

			log.Printf("Read error on FD %d: %v\n", fd, err)
			freeClient(loop, client)
			return
		}

		if n == 0 {
			log.Printf("Client on FD %d disconnected gracefully", fd)
			freeClient(loop, client)
			return
		}

		client.QueryBuf = client.QueryBuf[:len(client.QueryBuf)+n]

		processClientQueryBuffer(loop, client)
	}
}

// func SendReplyToClient(loop *EventLoop, fd int, clientData interface{}) {
// 	client := clientData.(*Client)

// 	if len(client.ReplyBuf) == 0 {
// 		loop.DeleteFileEvent(fd, unix.EPOLLOUT)
// 		return
// 	}

// 	n, err := unix.Write(fd, client.ReplyBuf)
// 	if err != nil {
// 		if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
// 			return
// 		}

// 		log.Printf("Write error on FD %d: %v\n", fd, err)
// 		freeClient(loop, client)
// 		return
// 	}

// 	client.ReplyBuf = client.ReplyBuf[n:]

// 	if len(client.ReplyBuf) == 0 {
// 		client.ReplyBuf = client.ReplyBuf[:0]
// 		loop.DeleteFileEvent(fd, unix.EPOLLOUT)
// 	}
// }

func processClientQueryBuffer(loop *EventLoop, client *Client) {

	cmds, bytesConsumed, err := readCommands(client.QueryBuf)

	if err != nil {
		log.Println("Parsing error:", err)
		freeClient(loop, client)
		return
	}

	if bytesConsumed > 0 {
		copy(client.QueryBuf, client.QueryBuf[bytesConsumed:])
		client.QueryBuf = client.QueryBuf[:len(client.QueryBuf)-bytesConsumed]
	}

	if len(cmds) == 0 {
		return
	}

	respond(cmds, client, loop)

	if len(client.ReplyBuf) > 0 {
		clientsPendingWrite[client.Fd] = client
	}

}
