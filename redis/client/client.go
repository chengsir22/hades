package client

import (
	"github.com/tidwall/redcon"
	"strings"
)

type cmdHandler func(cli *BitcaskClient, args [][]byte) (interface{}, error)

var supportedCommands = map[string]cmdHandler{
	"set":   set,
	"get":   get,
	"hset":  hset,
	"sadd":  sadd,
	"lpush": lpush,
	"zadd":  zadd,
}

func execClientCommand(conn redcon.Conn, cmd redcon.Command) {
	command := strings.ToLower(string(cmd.Args[0]))

	client, _ := conn.Context().(*BitcaskClient)
	switch command {
	case "quit":
		_ = conn.Close()
	case "ping":
		conn.WriteString("PONG")
	default:
		cmdFunc, ok := supportedCommands[command]
		if !ok {
			conn.WriteError("Err unsupported command: '" + command + "'")
			return
		}
		res, err := cmdFunc(client, cmd.Args[1:])
		if err != nil {
			if err == bitcask.ErrKeyNotFound {
				conn.WriteNull()
			} else {
				conn.WriteError(err.Error())
			}
			return
		}
		conn.WriteAny(res)
	}
}
