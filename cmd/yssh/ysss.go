package main

import (
	"YTools/ycomm"
	"fmt"
	"io"
	"log"
	"net"
	"os/exec"
	"strings"
	"time"
)

func doCommandExecute(conn net.Conn, str string) bool {
	split := strings.Split(str, " ")
	str = strings.ReplaceAll(str, "\n", "")
	str = strings.TrimSpace(str)
	head := split[0]
	arg := split[1:]
	fmt.Println("收到命令：" + str)
	// 执行命令
	cmd := exec.Command(head, arg...)
	output, err := cmd.CombinedOutput()
	fmt.Println("执行完成：" + str)
	if err != nil {
		fmt.Printf("Command execution error: %s\n", err)
		output = []byte(err.Error())
	}
	//fmt.Printf("combined out:\n%s\n", string(output))
	return ycomm.WriteByte1(conn, output)
}

func WriteSshMsg(conn net.Conn, msg string) bool {
	return ycomm.WriteByte1(conn, []byte(msg))
}

//服务端
func handleConnection(conn net.Conn) {
	defer conn.Close()

	for {
		// 从客户端接收命令
		byte0, err := ycomm.ReadByte1(conn)
		if err != nil {
			if err == io.EOF {
				fmt.Println("Connection closed by client")
			} else {
				fmt.Printf("Error reading command: %s\n", err)
			}
			return
		}

		str := string(byte0)
		exeOK := make(chan bool, 1)
		go func() {
			doCommandExecute(conn, str)
			exeOK <- true
		}()

		select {
		case <-exeOK:
			//fmt.Println("操作完成")

		case <-time.After(3 * time.Second):
			// 达到超时时间
			//fmt.Println("操作超时:", err)
			WriteSshMsg(conn, "执行超时")
		}

	}
}

func main() {
	// 监听端口
	listener, err := net.Listen("tcp", "0.0.0.0:8888")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	fmt.Println("Server started at 0.0.0.0:8888. Waiting for connections...")
	// 接受客户端连接并处理
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		fmt.Printf("New connection from: %s\n", conn.RemoteAddr())
		go handleConnection(conn)
	}
}
