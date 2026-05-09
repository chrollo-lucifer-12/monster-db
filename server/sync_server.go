package server

import (
	"fmt"
	"io"
	"strings"

	"github.com/redis-server/core"
	"github.com/redis-server/resp"
)

func toArrayString(ai []interface{}) ([]string, error) {
	as := make([]string, len(ai))
	for i := range ai {
		as[i] = ai[i].(string)
	}
	return as, nil
}

func readCommands(client io.ReadWriter) (core.RedisCmds, error) {
	var buf []byte = make([]byte, 512)

	n, err := client.Read(buf[:])
	if err != nil {
		return nil, err
	}

	values, err := resp.Decode(buf[:n])

	if err != nil {
		return nil, err
	}

	var cmds []*core.RedisCmd = make([]*core.RedisCmd, 0)

	for _, value := range values {
		tokens, err := toArrayString(value.([]interface{}))
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, &core.RedisCmd{
			Cmd:  strings.ToUpper(tokens[0]),
			Args: tokens[1:],
		})
	}

	return cmds, nil
}

func respondWithError(client io.ReadWriter, err error) {
	client.Write([]byte(fmt.Sprintf("-%s\r\n", err)))
}

func respond(cmds core.RedisCmds, client io.ReadWriter) {
	core.EvalAndInput(cmds, client)
}

// func RunSyncServer() {
// 	log.Println("starting the server on ", config.Host, config.Port)

// 	var con_clients int = 0

// 	lsnr, err := net.Listen("tcp", config.Host+":"+strconv.Itoa(config.Port))

// 	if err != nil {
// 		panic(err)
// 	}

// 	for {
// 		c, err := lsnr.Accept()
// 		if err != nil {
// 			panic(err)
// 		}
// 		con_clients += 1
// 		log.Println("client connected with address: ", c.RemoteAddr(), "concurrent clients: ", con_clients)

// 		for {
// 			cmd, err := readCommand(c)
// 			if err != nil {
// 				c.Close()
// 				con_clients -= 1
// 				log.Println("client disconnected with address: ", c.RemoteAddr(), "concurrent clients: ", con_clients)
// 				if err == io.EOF {
// 					break
// 				}
// 				log.Println("err", err)
// 			}

// 			respond(cmd, c)
// 		}
// 	}
// }
