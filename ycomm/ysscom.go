package ycomm

import (
	"errors"
	"fmt"
	"net"
)

//网络 流分隔符
//使用 ETX(3) 正文结束作为结束标志
const CLOSE_FLAG = byte(3)

const SERVER_PORT = "8888"

func ReadByte1(conn net.Conn) (rcvBytes []byte, err error) {
	tmpByte := make([]byte, 1)
	var cnt int = 0
	for {
		_, err := conn.Read(tmpByte)
		if err != nil {
			fmt.Println("错误(ERROR): ReadByte读取数据出错[", err, "]")
			//panic(err)
			err = errors.New("异常：" + err.Error())
			return nil, err
		}
		if tmpByte[0] == CLOSE_FLAG {
			break
		}

		rcvBytes = append(rcvBytes, tmpByte[0])
		cnt++
	}

	return rcvBytes, nil
}

func WriteByte1(conn net.Conn, b []byte) bool {
	b = append(b, CLOSE_FLAG)
	_, err := conn.Write(b)
	if err != nil {
		fmt.Printf("writeMsg【%s】失败\n", string(b))
		fmt.Println("错误(ERROR): WriteMsg发送数据出错[", err, "]")
		return false
	}
	return true
}
