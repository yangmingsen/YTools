package ynet

import (
	"YTools/ycomm"
	"fmt"
	"net"
)

//如果绑定失败 返回nil
func GetBindNet(ip string, port string) (bindIp net.Listener) {
	bindIp, err0 := net.Listen("tcp", ip+":"+port)
	if err0 == nil {
		return bindIp
	} else {
		fmt.Println("ip: ", ip, " Can't bind, [", err0, "]")
		return nil
	}
}

func GetBindTcpListenNet(ip, port string) net.Listener {
	return GetBindNet(ip, port)
}

func GetRemoteConnection(ip, port string) (conn net.Conn, err1 error) {
	ip = ip + ":" + port
	conn, err1 = net.Dial("tcp", ip)

	return conn, err1
}

//响应
func SendResponse(conn net.Conn, res ycomm.ResponseInfo) {
	//发送请求处理完毕信息
	resStr := ycomm.ParseResponseToJsonStr(res)
	ycomm.WriteMsg(conn, resStr)
}

//发送请求
func SendRequest(conn net.Conn, req ycomm.RequestInfo) {
	ycomm.WriteMsg(conn, req.ParseToJsonStr())
}
