package server

import (
	"log"
	"net"

	"runtime"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/redis-server/config"
	"github.com/redis-server/core"
	"golang.org/x/sys/unix"
)

var (
	zmalloc                           = NewZAllocator(512 * 1024 * 1024)
	clientsPendingWrite               = make(map[int]*Client)
	con_clients                       = 0
	lastRDB             time.Time     = time.Now()
	rdbFrequency        time.Duration = 900 * time.Second
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

	qBuf, err := zmalloc.Alloc(4096)
	if err != nil {
		log.Println("OOM")
		return nil
	}

	rBuf, err := zmalloc.Alloc(4096)
	if err != nil {
		log.Println("OOM")
		zmalloc.Free(qBuf)
		return nil
	}

	return &Client{
		Fd:       fd,
		QueryBuf: qBuf[:0],
		ReplyBuf: rBuf[:0],
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

		if err := unix.SetNonblock(fd, true); err != nil {
			log.Println("SerNonBlock error :", err)
			unix.Close(fd)
			continue
		}

		if err := unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_NODELAY, 1); err != nil {
			log.Println("Set TCP_NODELAY error :", err)
		}

		client := NewClient(fd)

		err = el.AddFileEvent(fd, unix.EPOLLIN, ReadQueryFromClient, client)
		if err != nil {
			log.Println("Failed to add client to epoll: ", err)
			unix.Close(fd)
			continue
		}

		con_clients++
	}
}

func freeClient(loop *EventLoop, client *Client) {
	loop.DeleteFileEvent(client.Fd, unix.EPOLLIN|unix.EPOLLOUT)
	unix.Close(client.Fd)
	con_clients--

	zmalloc.Free(client.QueryBuf)
	zmalloc.Free(client.ReplyBuf)
}

func ReadQueryFromClient(loop *EventLoop, fd int, clientData interface{}) {
	client := clientData.(*Client)

	for {

		if len(client.QueryBuf) == cap(client.QueryBuf) {
			newCap := cap(client.QueryBuf) * 2
			if newCap == 0 {
				newCap = 4096
			}

			newBuf, err := zmalloc.Alloc(newCap)
			if err != nil {
				log.Println("OOM: Client sent too much data, disconnecting")
				freeClient(loop, client)
				return
			}

			newBuf = newBuf[:len(client.QueryBuf)]
			copy(newBuf, client.QueryBuf)

			zmalloc.Free(client.QueryBuf)

			client.QueryBuf = newBuf
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
			//	log.Printf("Client on FD %d disconnected gracefully", fd)
			freeClient(loop, client)
			return
		}

		client.QueryBuf = client.QueryBuf[:len(client.QueryBuf)+n]

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
		client.ReplyBuf = client.ReplyBuf[:0]
		loop.DeleteFileEvent(fd, unix.EPOLLOUT)
	}
}

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

	writerWrapper := ReplyBufferWrapper{client: client}

	respond(cmds, writerWrapper)

	if len(client.ReplyBuf) > 0 {
		clientsPendingWrite[client.Fd] = client
	}

	// loop.AddFileEvent(client.Fd, unix.EPOLLOUT, SendReplyToClient, client)
}

func RunAsyncServer() error {

	go func() {
		log.Println("Starting pprof server on http://localhost:6060")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

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
