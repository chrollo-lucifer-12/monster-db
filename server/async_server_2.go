package server

import (
	"log"
	"net"
	"runtime"
	"time"

	"github.com/redis-server/config"
	"github.com/redis-server/core"
	"golang.org/x/sys/unix"
)

var (
	con_clients                = 0
	lastRDB      time.Time     = time.Now()
	rdbFrequency time.Duration = 900 * time.Second
)

type Client struct {
	Fd       int
	QueryBuf []byte
	ReplyBuf []byte
}

type ReplyBufferWrapper struct {
	client *Client
}

func (w ReplyBufferWrapper) Write(p []byte) (n int, err error) {
	w.client.ReplyBuf = append(w.client.ReplyBuf, p...)
	return len(p), nil
}

func (w ReplyBufferWrapper) Read(p []byte) (n int, err error) {
	return 0, nil
}

func NewClient(fd int) *Client {
	return &Client{
		Fd:       fd,
		QueryBuf: make([]byte, 0),
		ReplyBuf: make([]byte, 0),
	}
}

func ServerCronHandler(loop *EventLoop, id int64, clientData interface{}) int {
	now := time.Now()

	if now.Sub(lastRDB) > rdbFrequency {
		core.TriggerRDB()
		lastRDB = now
	}

	core.DeleteExpiredKey()

	return 1000
}

func AcceptTcpHandler(el *EventLoop, serverFD int, clientData interface{}) {

	for {
		fd, _, err := unix.Accept(serverFD)
		if err != nil {
			if err == unix.EAGAIN || err == unix.EWOULDBLOCK {

				break
			}

			if err == unix.EMFILE || err == unix.ENFILE {

				break
			}

			log.Println("Accept Error: ", err)
			return
		}

		con_clients++

		if err := unix.SetNonblock(fd, true); err != nil {
			log.Println("SerNonBlock error :", err)
			unix.Close(fd)
			con_clients--
			continue
		}

		client := NewClient(fd)

		err = el.AddFileEvent(fd, unix.EPOLLIN, ReadQueryFromClient, client)
		if err != nil {
			log.Println("Failed to add client to epoll: ", err)
			unix.Close(fd)
			con_clients--
			continue
		}
	}
}

func freeClient(loop *EventLoop, client *Client) {
	loop.DeleteFileEvent(client.Fd, unix.EPOLLIN|unix.EPOLLOUT)
	unix.Close(client.Fd)
	con_clients--
}

func ReadQueryFromClient(loop *EventLoop, fd int, clientData interface{}) {
	client := clientData.(*Client)

	readBuffer := make([]byte, 4096)

	for {
		n, err := unix.Read(fd, readBuffer)
		if err != nil {

			if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
				break
			}

			log.Printf("Read error on FD %d: %v\n", fd, err)
			freeClient(loop, client)
			return
		}

		if n == 0 {
			//	log.Printf("Client on FD %d disconnected gracefully", fd)
			freeClient(loop, client)
			return
		}

		client.QueryBuf = append(client.QueryBuf, readBuffer[:n]...)

		processClientQueryBuffer(loop, client)
	}
}

func SendReplyToClient(loop *EventLoop, fd int, clientData interface{}) {
	client := clientData.(*Client)

	if len(client.ReplyBuf) == 0 {
		loop.DeleteFileEvent(fd, unix.EPOLLOUT)
		return
	}

	n, err := unix.Write(fd, client.ReplyBuf)
	if err != nil {
		if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
			return
		}

		log.Printf("Write error on FD %d: %v\n", fd, err)
		freeClient(loop, client)
		return
	}

	client.ReplyBuf = client.ReplyBuf[n:]

	if len(client.ReplyBuf) == 0 {
		loop.DeleteFileEvent(fd, unix.EPOLLOUT)
	}
}

func processClientQueryBuffer(loop *EventLoop, client *Client) {

	cmds, err := readCommands(client.QueryBuf)

	if err != nil {
		log.Println("Parsing error:", err)
		freeClient(loop, client)
		return
	}

	if len(cmds) == 0 {
		return
	}

	client.QueryBuf = client.QueryBuf[:0]

	writerWrapper := ReplyBufferWrapper{client: client}

	respond(cmds, writerWrapper)

	loop.AddFileEvent(client.Fd, unix.EPOLLOUT, SendReplyToClient, client)
}

func RunAsyncServer2() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	log.Println("Starting server on ", config.Host, config.Port)

	maxClients := 100000
	loop, err := CreateEventLoop(maxClients)
	if err != nil {
		return err
	}
	defer unix.Close(loop.EpollFD)

	serverFD, err := unix.Socket(unix.AF_INET, unix.O_NONBLOCK|unix.SOCK_STREAM, 0)
	if err != nil {
		return err
	}
	defer unix.Close(serverFD)

	unix.SetsockoptInt(serverFD, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)

	ip4 := net.ParseIP(config.Host)
	err = unix.Bind(serverFD, &unix.SockaddrInet4{
		Port: config.Port,
		Addr: [4]byte{ip4[0], ip4[1], ip4[2], ip4[3]},
	})
	if err != nil {
		return err
	}

	if err = unix.Listen(serverFD, maxClients); err != nil {
		return err
	}

	err = loop.AddFileEvent(serverFD, unix.EPOLLIN, AcceptTcpHandler, nil)
	if err != nil {
		return err
	}

	loop.AddTimeEvent(1000, ServerCronHandler, nil)

	loop.Main()

	return nil
}
