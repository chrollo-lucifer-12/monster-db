package server

import (
	"log"
	"net"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/redis-server/config"
	"golang.org/x/sys/unix"
)

func RunGoServer(wg *sync.WaitGroup) error {
	defer wg.Done()

	defer func() {
		atomic.StoreInt32(&eStatus, EnngineStatus_SHUTTING_DOWN)
	}()

	log.Println("Starting server on", config.Host, config.Port)

	maxClients := 100000

	serverFD, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0)
	if err != nil {
		return err
	}
	defer unix.Close(serverFD)

	if err := unix.SetsockoptInt(
		serverFD,
		unix.SOL_SOCKET,
		unix.SO_REUSEADDR,
		1,
	); err != nil {
		return err
	}

	ip4 := net.ParseIP(config.Host).To4()

	err = unix.Bind(serverFD, &unix.SockaddrInet4{
		Port: config.Port,
		Addr: [4]byte{ip4[0], ip4[1], ip4[2], ip4[3]},
	})
	if err != nil {
		return err
	}

	if err := unix.Listen(serverFD, maxClients); err != nil {
		return err
	}

	numWorkers := runtime.NumCPU()

	clients := make(chan *Client, runtime.NumCPU()*4)

	for i := 0; i < numWorkers; i++ {
		go clientWorker(clients)
	}

	for atomic.LoadInt32(&eStatus) != EnngineStatus_SHUTTING_DOWN {
		fd, _, err := unix.Accept(serverFD)
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}

		if err := unix.SetsockoptInt(
			fd,
			unix.IPPROTO_TCP,
			unix.TCP_NODELAY,
			1,
		); err != nil {
			log.Println("Set TCP_NODELAY error:", err)
		}

		client := NewClient(fd)

		con_clients++

		clients <- client
	}

	return nil
}

func clientWorker(clients <-chan *Client) {
	for client := range clients {
		handleClient(client)
	}
}

func handleClient(client *Client) {
	defer func() {
		unix.Close(client.Fd)
		con_clients--
	}()

	buf := make([]byte, 4096)

	for {
		if (client.flag & CLIENT_BLOCKED) != 0 {
			continue
		}

		n, err := unix.Read(client.Fd, buf)

		if err != nil {
			log.Printf("Read error on FD %d: %v", client.Fd, err)
			return
		}

		if n == 0 {
			return
		}

		client.QueryBuf = append(client.QueryBuf, buf[:n]...)

		for {
			cmds, bytesConsumed, err := readCommands(client.QueryBuf)

			if err != nil {
				log.Println("Parsing error:", err)
				return
			}

			if len(cmds) == 0 {
				break
			}

			copy(
				client.QueryBuf,
				client.QueryBuf[bytesConsumed:],
			)

			client.QueryBuf =
				client.QueryBuf[:len(client.QueryBuf)-bytesConsumed]

			respond(cmds, client, nil)

			if len(client.ReplyBuf) > 0 {
				if err := writeReply(client); err != nil {
					return
				}
			}
		}
	}
}

func writeReply(client *Client) error {
	for len(client.ReplyBuf) > 0 {
		n, err := unix.Write(client.Fd, client.ReplyBuf)

		if err != nil {
			log.Printf(
				"Write error on FD %d: %v",
				client.Fd,
				err,
			)
			return err
		}

		client.ReplyBuf = client.ReplyBuf[n:]
	}

	client.ReplyBuf = client.ReplyBuf[:0]

	return nil
}
