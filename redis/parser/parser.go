package main

import (
	"bufio"
	"fmt"
	"net"
)

// redis 协议解析的示例
func main() {
	conn, err := net.Dial("tcp", "localhost:6379")
	if err != nil {
		panic(err)
	}

	// 向 Redis 发送一个命令
	cmd := "set k-name-2 bitcask-kv-2\r\n"
	conn.Write([]byte(cmd))

	// 解析 Redis 响应
	reader := bufio.NewReader(conn)
	res, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	fmt.Println(res)
}
