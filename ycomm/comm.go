package ycomm

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
)

//单文件变量
//======================================================================
const (
	SizeB  int64 = 1024
	SizeKB int64 = 1048576
	SizeMB int64 = 1073741824
	SizeGB int64 = 1099511627776
)

const (
	B  = 1
	KB = 2
	MB = 3
	GB = 4
)

var debug bool

//多文件变量
//======================================================================
type CFileInfo struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
	Path  string `json:"path"`
}

type ResponseInfo struct {
	Ok      bool   `json:"ok"`
	Message string `json:"message"`

	//用于自定义通信规则时使用 其他不用是 传 OK
	Status string `json:"status"`
}

const msgCloseFlag = "\r"

func ParseCFileInfoToJsonStr(info CFileInfo) string {
	bytes, _ := json.Marshal(info)
	return string(bytes)
}

func ParseStrToCFileInfo(str string) CFileInfo {
	return ParseByteToCFileInfo([]byte(str))
}

func ParseByteToCFileInfo(bytes []byte) CFileInfo {
	var result CFileInfo

	json.Unmarshal(bytes, &result)
	return result
}

func ParseResponseToJsonStr(res ResponseInfo) string {
	bytes, _ := json.Marshal(res)
	return string(bytes)
}

func ParseStrToResponseInfo(str string) ResponseInfo {
	return ParseByteToResponseInfo([]byte(str))
}

func ParseByteToResponseInfo(bytes []byte) ResponseInfo {
	var result ResponseInfo

	json.Unmarshal(bytes, &result)
	return result
}

//发送数据
func WriteMsg(conn net.Conn, msg string) bool {
	sendMsg := []byte(msg + msgCloseFlag) // 13
	_, err := conn.Write(sendMsg)
	if err != nil {
		fmt.Printf("writeMsg【%s】失败\n", sendMsg)
		panic(err)
		return false
	}
	//fmt.Println("发送数据【"+msg+"】长度=", wLen)
	return true
}

//以 \r 结尾
func ReadMsg(conn net.Conn) string {
	var rcvBytes []byte

	tmpByte := make([]byte, 1)
	var cnt int = 0

	for {
		_, err := conn.Read(tmpByte)
		if err != nil {
			panic(err)
			break
		}
		if tmpByte[0] == byte(13) {
			break
		}

		rcvBytes = append(rcvBytes, tmpByte[0])
		cnt++
	}

	return string(rcvBytes[:cnt])

}

func isIncludeIP(ip string) bool {
	excludeList := [...]string{"127", "local"}
	for _, ex := range excludeList {
		if strings.HasPrefix(ip, ex) == true {
			return true
		}
	}
	return false
}

// 获取所有ipv4绑定列表
func GetLocalIpv4List() []string {
	addrs, err := net.InterfaceAddrs() //获取所有ip地址, 包含ipv4,ipv6
	if err != nil {
		panic(err)
	}

	//addrsLen := len(addrs);

	res := make([]string, 0)

	//fmt.Println(addrs)
	for _, addr := range addrs {
		//fmt.Println(addr)
		ip := addr.String()
		contains := strings.Contains(ip, ".")
		if contains {

			if isIncludeIP(ip) == false {
				sprIP := strings.Split(ip, "/")
				res = append(res, sprIP[0])
			}
		}
	}
	//fmt.Println(res)
	return res
}
