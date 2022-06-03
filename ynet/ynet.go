package ynet

import (
	"YTools/ycomm"
	"YTools/ylog"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"
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

func ServerSocket(port string) (conn net.Listener, err error) {
	return net.Listen("tcp", ycomm.LOCAL_HOST+":"+port)
}

func Socket(ip, port string) (conn net.Conn, err error) {
	return net.Dial("tcp", ip+":"+port)
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

//网络流转发
func TransferStream(from, to net.Conn) {
	go DoTransferStream(from, to)
	go DoTransferStream(to, from)
}

func DoTransferStream(from, to net.Conn) {

	from.SetWriteDeadline(time.Now().Add(10 * time.Second))
	to.SetWriteDeadline(time.Now().Add(15 * time.Second))

	buf := make([]byte, 512)
	for {
		rn, err0 := from.Read(buf)
		if err0 != nil {
			ylog.Logf("TransferStream读取网络流出错>>>>", err0)
			break
		}

		_, err1 := to.Write(buf[:rn])
		if err1 != nil {
			ylog.Logf("TransferStream写入网络流出错>>>>", err1)
			break
		}

	}
}

// relay copies between left and right bidirectionally
func relay(left, right net.Conn) error {
	var err, err1 error
	var wg sync.WaitGroup
	var wait = 5 * time.Second
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err1 = io.Copy(right, left)
		right.SetReadDeadline(time.Now().Add(wait)) // unblock read on right
	}()
	_, err = io.Copy(left, right)
	left.SetReadDeadline(time.Now().Add(wait)) // unblock read on left
	wg.Wait()
	if err1 != nil && !errors.Is(err1, os.ErrDeadlineExceeded) { // requires Go 1.15+
		return err1
	}
	if err != nil && !errors.Is(err, os.ErrDeadlineExceeded) {
		return err
	}
	return nil
}

func GetRemoteRegInfo(detectIp string) []ycomm.YrecvBase {
	conn, err0 := GetRemoteConnection(detectIp, ycomm.RoutePort)
	defer conn.Close()
	if err0 != nil {
		ylog.Logf("网络连接>>>", detectIp+":"+ycomm.RoutePort, ">>>>异常[", err0, "]>>>doRouteDect连接失败====>退出")
		return nil
	}

	req := ycomm.RequestInfo{
		Cmd:   ycomm.YDECT_MSG,
		Data:  "msg:no",
		Other: "",
	}

	// WriteMsg(conn, req.ParseToJsonStr())
	//发起请求
	SendRequest(conn, req)
	ylog.Logf("向Route发送数据>>>>", req.ParseToJsonStr())

	//接收响应数据
	byte0, _ := ycomm.ReadByte0(conn)
	resInfo := ycomm.ParseByteToResponseInfo(byte0)
	ylog.Logf("获取到响应数据>>>", ycomm.ParseResponseToJsonStr(resInfo))

	list := ycomm.ParseStrToYrecvBaseList(resInfo.Message)

	return list
}
