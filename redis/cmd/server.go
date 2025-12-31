package main

import (
	bitcask "bitcask-go"
	bitcask_redis "bitcask-go/redis"
	"log"
	"sync"

	"github.com/tidwall/redcon"
)

var addr = "127.0.0.1:6380"

type BitcaskServer struct {
	server *redcon.Server
	dbs    map[int]*bitcask_redis.RedisDataStructure
	mu     sync.RWMutex
}

func main() {
	// 打开数据库
	db, err := bitcask_redis.NewRedisDataStructure(bitcask.DefaultOptions)
	if err != nil {
		panic(err)
	}

	// 初始化 server
	bitcaskServer := &BitcaskServer{
		dbs: make(map[int]*bitcask_redis.RedisDataStructure),
	}
	bitcaskServer.dbs[0] = db
	bitcaskServer.server = redcon.NewServer(addr, execClientCommand, bitcaskServer.accept, bitcaskServer.close)
	bitcaskServer.listen()
}

func (svr *BitcaskServer) listen() {
	log.Println("bitcask engine running, ready to accept connections")
	_ = svr.server.ListenAndServe()
}

func (svr *BitcaskServer) accept(conn redcon.Conn) bool {
	cli := new(BitcaskClient)
	svr.mu.Lock()
	defer svr.mu.Unlock()
	cli.server = svr
	cli.db = svr.dbs[0]
	conn.SetContext(cli)
	return true
}

func (svr *BitcaskServer) close(conn redcon.Conn, err error) {
	for _, db := range svr.dbs {
		_ = db.Close()
	}
	_ = svr.server.Close()
	log.Println("bitcask engine exit...")
}

// redis 协议解析示例
//func main() {
//	conn, err := net.Dial("tcp", "localhost:6379")
//	if err != nil {
//		panic(err)
//	}
//	defer conn.Close()
//
//	// 发送一个命令
//	cmd := "set kv bitcask-storage-yyds\r\n"
//	conn.Write([]byte(cmd))
//
//	// 解析响应
//	reader := bufio.NewReader(conn)
//	res, err := reader.ReadString('\n')
//	fmt.Println(res, err)
//}
