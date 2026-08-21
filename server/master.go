package server

import (
	"fmt"
	"log"
	"net"

	"golang.org/x/sys/unix"
)

type Replica struct {
	FD     int
	Client *Client
}

var MasterFD int = -1
var replicas []*Replica

func AddReplica(loop *EventLoop, host string, port int) error {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0)
	if err != nil {
		return err
	}

	ip := net.ParseIP(host).To4()
	if ip == nil {
		unix.Close(fd)
		return fmt.Errorf("invalid IP address: %s", host)
	}

	addr := &unix.SockaddrInet4{
		Port: port,
	}
	copy(addr.Addr[:], ip)

	if err := unix.Connect(fd, addr); err != nil {
		unix.Close(fd)
		return err
	}

	if _, err := unix.Write(fd, []byte("*1\r\n$7\r\nREPLICA\r\n")); err != nil {
		unix.Close(fd)
		return err
	}

	buf := make([]byte, 64)

	n, err := unix.Read(fd, buf)
	if err != nil {
		unix.Close(fd)
		return err
	}

	if string(buf[:n]) != "+OK\r\n" {
		unix.Close(fd)
		return fmt.Errorf("master rejected replica: %q", buf[:n])
	}

	if err := unix.SetNonblock(fd, true); err != nil {
		unix.Close(fd)
		return err
	}

	MasterFD = fd

	client := NewClient(fd)

	loop.AddFileEvent(
		fd,
		unix.EPOLLIN,
		ReadQueryFromClient,
		client,
	)

	return nil
}

func RegisterReplica(client *Client) {
	client.IsReplica = true

	replicas = append(replicas, &Replica{
		FD:     client.Fd,
		Client: client,
	})
}

func Replicate(cmd []byte) {
	for i := 0; i < len(replicas); {
		replica := replicas[i]

		_, err := unix.Write(replica.FD, cmd)

		if err != nil {
			log.Println(err)

			unix.Close(replica.FD)

			replicas[i] = replicas[len(replicas)-1]
			replicas = replicas[:len(replicas)-1]
			continue
		}

		i++
	}
}
