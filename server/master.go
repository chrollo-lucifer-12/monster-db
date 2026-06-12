package server

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

var (
	MasterRunID        string = "master_run_id_777777777777777777777777777777"
	MasterGlobalOffset int64  = 0

	ReplBacklog         = make([]byte, 0, 1024*1024)
	BacklogMaxCap int64 = 1024 * 1024

	ConnectedReplicas = make(map[int]*Client)
)

func HandleInformReplicas(cmd []byte) {
	if len(cmd) == 0 {
		return
	}

	MasterGlobalOffset += int64(len(cmd))

	ReplBacklog = append(ReplBacklog, cmd...)
	if int64(len(ReplBacklog)) > BacklogMaxCap {
		ReplBacklog = ReplBacklog[int64(len(ReplBacklog))-BacklogMaxCap:]
	}

	for fd, replicaClient := range ConnectedReplicas {
		replicaClient.ReplyBuf = append(replicaClient.ReplyBuf, cmd...)

		clientsPendingWrite[fd] = replicaClient
	}
}

func HandleMasterReplConfCommand(c *Client, args []string) {
	if len(args) < 2 {
		c.ReplyBuf = append(c.ReplyBuf, []byte("-ERR syntax error\r\n")...)
		return
	}

	subCommand := strings.ToLower(args[0])

	switch subCommand {
	case "listening-port", "capa":

		c.ReplyBuf = append(c.ReplyBuf, []byte("+OK\r\n")...)

	case "ack":

		replicaOffset, err := strconv.ParseInt(args[1], 10, 64)
		if err == nil {

			lag := MasterGlobalOffset - replicaOffset
			if lag > 0 {
				log.Printf("Replica on FD %d tracking with a lag of %d bytes\n", c.Fd, lag)
			}
		}

	}
}

func HandleMasterPsyncCommand(loop *EventLoop, c *Client, args []string) {
	if len(args) < 2 {
		c.ReplyBuf = append(c.ReplyBuf, []byte("-ERR syntax error\r\n")...)
		return
	}

	reqReplID := args[0]
	reqOffset, _ := strconv.ParseInt(args[1], 10, 64)

	diff := MasterGlobalOffset - int64(len(ReplBacklog))

	if reqReplID == MasterRunID && reqOffset >= diff && reqOffset <= MasterGlobalOffset {
		log.Printf("Replica on FD %d qualified for Partial Resync (+CONTINUE)\n", c.Fd)
		c.ReplyBuf = append(c.ReplyBuf, []byte("+CONTINUE\r\n")...)

		deltaStart := reqOffset - diff
		if deltaStart < int64(len(ReplBacklog)) {
			c.ReplyBuf = append(c.ReplyBuf, ReplBacklog[deltaStart:]...)
		}
	} else {

		log.Printf("Replica on FD %d requires Full Resync (+FULLRESYNC)\n", c.Fd)
		fullResyncHeader := fmt.Sprintf("+FULLRESYNC %s %d\r\n", MasterRunID, MasterGlobalOffset)
		c.ReplyBuf = append(c.ReplyBuf, []byte(fullResyncHeader)...)

		mockSnapshot := []byte("*3\r\n$3\r\nSET\r\n$1\r\nk\r\n$1\r\nv\r\n")
		c.ReplyBuf = append(c.ReplyBuf, mockSnapshot...)
	}

	ConnectedReplicas[c.Fd] = c
	clientsPendingWrite[c.Fd] = c

	loop.AddFileEvent(c.Fd, unix.EPOLLIN, HandleLiveReplicaTraffic, c)
}

func HandleLiveReplicaTraffic(loop *EventLoop, fd int, clientData interface{}) {
	client := clientData.(*Client)

	var buf [1024]byte
	n, err := unix.Read(fd, buf[:])
	if err != nil || n <= 0 {
		log.Printf("Replica connection on FD %d dropped. Removing from stream feed arrays.\n", fd)

		delete(ConnectedReplicas, fd)
		cleanupClient(client, loop)
		return
	}

	client.QueryBuf = append(client.QueryBuf, buf[:n]...)

	input := string(client.QueryBuf)
	if strings.Contains(input, "REPLCONF") && strings.Contains(input, "ACK") {

		client.QueryBuf = client.QueryBuf[:0]
	}
}
