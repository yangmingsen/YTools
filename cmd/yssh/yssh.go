package main

import (
	"YTools/ycomm"
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

var showUser string
var exitFlag = make(chan bool, 1)

//发送命令
func sendCommand(conn net.Conn, cmd string) bool {
	sendBytes := []byte(cmd)
	return ycomm.WriteByte1(conn, sendBytes)
}

//读取用户输入的命令
func inputCommand(reader *bufio.Reader) (string, bool) {
	// 读取用户输入的命令
	//fmt.Print("[yms@yms]$ ")
	command, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("获取输入命令出错: ", err)
		return "", false
	}
	command = strings.TrimSpace(command)
	if command == "" && len(command) == 0 {
		//空字符情况处理
		return "", false
	}

	return command, true
}

//接收服务响应
func getServerResponse(conn net.Conn) bool {
	byte1, err := ycomm.ReadByte1(conn)
	if err != nil {
		fmt.Printf("收到错误响应: %s\n", err)
		return false
	}
	str := string(byte1)
	fmt.Println(str)

	return true
}

func recvServerResponse(conn net.Conn) {
	for {
		isOK := getServerResponse(conn)
		fmt.Print(showUser)
		if !isOK {
			exitFlag <- true
		}
	}
}

var flags struct {
	remoteIP string
	user     string
}

func getUsage() {
	flag.Usage()
	fmt.Println("例如: yssh -d 目标地址 -u 用户名")
}

func main() {
	flag.StringVar(&flags.remoteIP, "d", "", "目标ip")
	flag.StringVar(&flags.user, "u", "who", "使用用户")
	flag.Parse()

	if flags.remoteIP == "" {
		getUsage()
		return
	}
	showUser = "[" + flags.user + "@" + flags.remoteIP + "]$ "

	address := flags.remoteIP + ":" + ycomm.SERVER_PORT
	// 连接到服务器
	conn, err := net.Dial("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	go recvServerResponse(conn)
	go func() {
		fmt.Print(showUser)
		reader := bufio.NewReader(os.Stdin)
		for {
			// 读取用户输入的命令
			command, ok := inputCommand(reader)
			if !ok {
				continue
			}

			// 发送命令到服务器
			sendOK := sendCommand(conn, command)
			if !sendOK {
				continue
			}
		}
	}()

	<-exitFlag
	fmt.Println("结束进程")
}
