package main

import (
	"YTools/ycomm"
	"YTools/ylog"
	"YTools/ynet"
	"flag"
	"fmt"
	"io"
	"net"
	"strconv"
)

var yRouteExit = make(chan bool)
var bindPort = ycomm.RoutePort

var flags struct {
	ListenIP string
}

func routeExit() {
	yRouteExit <- true
}

func getUsage() {
	flag.Usage()
	fmt.Println("例如: yrout")
	fmt.Println("例如: yrout -b 10.3.4.2")
}

//ydect 报文请求及响应

//yrecv请求信息及响应
//注册信息 RequestInfo{cmd: "yrecv_init", data: "name:yms ip:192.168.25.88 cpu:8", other:""}
//	响应信息ResponseInfo{ok: true, message:"Recvive", status:"OK"}

//计数
var YrecvRegInfoNum = 0

//注册结构
type YrecvRegInfo struct {
	ycomm.YrecvBase
	CanConn    bool       //route是否可用直接连接yrecv
	BaseConn   net.Conn   //基础连接用于通信使用
	SingleConn net.Conn   //用于当CanConn=false时,单文件传输通道
	mulNum     int        //由ysend请求信息确定
	MultiConn  []net.Conn //用于当CanConn=false时,多文件传输通道,数量由mulNum确定
}

var yrecvRegInfo = make(map[string]YrecvRegInfo)

//连通性检查
func checkCanConnect(name string) {

	var recvReg = yrecvRegInfo[name]

	ylog.Logf("连通性检查>>>>连接ip=>", recvReg.Ip)
	conn, err0 := ynet.GetRemoteConnection(recvReg.Ip, ycomm.MultiRemotePort)
	if err0 != nil {
		ylog.Logf("Route直连Yrecv异常>>>[", err0, "]")
		//recvReg.CanConn = false
	} else {
		ycomm.WriteMsg(conn, ycomm.YROUTE_CHECK_YRECV)
		ylog.Logf("向yrecv>>>端口", ycomm.MultiRemotePort, ">>>发送check命令")

		byte0, _ := ycomm.ReadByte0(conn)
		res := ycomm.ParseByteToResponseInfo(byte0)

		ylog.Logf("收到来自yrecv的连通性检查响应数据>>>>[", res, "]")
		if res.Ok {
			ylog.Logf("可以连通>>>>>>")

			recvReg.CanConn = true
			yrecvRegInfo[name] = recvReg
		}
	}

	tmp := yrecvRegInfo[name]
	ylog.Logf("修改后值===", tmp)

	conn.Close()
}

func doYrecvEvent(conn net.Conn) {
	for {
		msgByte, err0 := ycomm.ReadByte0(conn)
		if err0 != nil {
			ylog.Logf("doYrecvEnvent出现异常,错误>>>>", err0, ">>>结束")
			break
		}
		msgStr := string(msgByte)
		ylog.Logf("收到yrecv消息>>>>>", msgStr)

	}
}

func doTransferStream(fileSize int64, from, to net.Conn) {
	defer from.Close()
	defer to.Close()

	//开始转发流数据
	buf := make([]byte, 4096)
	var cur int64 = 0
	for {
		rn, err0 := from.Read(buf)
		to.Write(buf[:rn])
		cur += int64(rn)

		if cur == fileSize {
			ylog.Logf("传输完毕>>>>>")
			break
		}
		if err0 != nil && err0 != io.EOF {
			fmt.Println("读取网络流出错>>>>[", err0, "]")
			break
		}

	}
}

//处理单文件路由传输
func doSingleFileHandler(requestInfo ycomm.RequestInfo, ysendConn net.Conn) {
	//单文件传输
	//ysend 请求信息
	//RequestInfo{cmd: "y_s_s_f", data: "name/ayms/nfileName/ahi.txt/nfileSize/a2342" , other:"" }

	//解析body数据
	dMap := ycomm.ParseStrToMapData(requestInfo.Data)
	ylog.Logf("解析ysend的body数据>>>>>", dMap)

	//1.数据检验
	var cName string
	if name, ok := dMap[ycomm.SEND_TO_NAME]; ok == false {
		ylog.Logf(">>>>ysend发送数据有误[name不存在]>>>>")
		fmt.Println(">>>>ysend发送数据有误[name不存在]>>>>", requestInfo.ParseToJsonStr())

		ynet.SendResponse(ysendConn, ycomm.ResponseInfo{Ok: false, Message: "name不存在", Status: "no"})

		ysendConn.Close()
	} else {
		cName = name
	}
	ylog.Logf("的body数据>>>>>存在", ycomm.SEND_TO_NAME)

	var fileSize int64
	if fsr, ok := dMap[ycomm.FILE_SIZE]; ok {
		fileSizet, _ := strconv.ParseInt(fsr, 10, 64)
		fileSize = fileSizet
	} else {
		ylog.Logf(">>>>ysend发送数据有误fileSize不存在]>>>>")
		fmt.Println(">>>>ysend发送数据有误[fileSize不存在]>>>>")

		ynet.SendResponse(ysendConn, ycomm.ResponseInfo{Ok: false, Message: "fileSize不存在", Status: "no"})

		ysendConn.Close()
	}
	ylog.Logf("的body数据>>>>>存在", ycomm.FILE_SIZE)

	if _, ok := dMap[ycomm.FILE_NAME]; ok == false {
		ylog.Logf(">>>>ysend发送数据有误fileName不存在]>>>>")
		fmt.Println(">>>>ysend发送数据有误[fileName不存在]>>>>")

		ynet.SendResponse(ysendConn, ycomm.ResponseInfo{Ok: false, Message: "fileName不存在", Status: "no"})

		ysendConn.Close()
	}
	ylog.Logf("的body数据>>>>>存在", ycomm.FILE_NAME)

	if regInfo, ok := yrecvRegInfo[cName]; ok {
		ylog.Logf("已匹配>>>", yrecvRegInfo[cName], ">>>>准备连接")
		if regInfo.CanConn { //溪流模式
			ylog.Logf("溪流模式开始连接yrecv>>>>>", regInfo.Ip, ">>>>")
			yrecvConn, err0 := ynet.GetRemoteConnection(regInfo.Ip, ycomm.MultiRemotePort)
			if err0 == nil { //如果连接成功
				//向yrecv发送命令
				ycomm.WriteMsg(yrecvConn, ycomm.YROUTE_SEND_SINGLE_FILE)

				//发送文件大小信息给yrecv
				var sendMap = make(map[string]string)
				sendMap["fileName"] = dMap["fileName"]
				sendMap["fileSize"] = dMap["fileSize"]
				sendStr := ycomm.ParseMapToStr(sendMap)
				ylog.Logf("发送文件大小信息>>>>", sendStr)
				ycomm.WriteMsg(yrecvConn, sendStr)

				//等待yrecv的响应
				rcvMsg := ycomm.ReadMsg(yrecvConn)
				ylog.Logf("收到yrecv响应>>>>", rcvMsg)
				resp := ycomm.ParseStrToResponseInfo(rcvMsg)
				if resp.Ok {
					//发送ok给ysend
					ynet.SendResponse(ysendConn, ycomm.ResponseInfo{Ok: true, Message: "ok prepare fileSteam", Status: "ok"})
					ylog.Logf("准备完毕...开始. ysend=>yroute=>yrecv")

					doTransferStream(fileSize, ysendConn, yrecvConn)
				}

			} else { //如果连接yrecv失败
				ylog.Logf("开始连接yrecv>>>>>", regInfo.Ip, ">>>>")

				ynet.SendResponse(ysendConn, ycomm.ResponseInfo{Ok: false, Message: "connect yrecv failed", Status: "no"})
			}
		} else { //大海模式
			ylog.Logf("启动大海模式>>>>>")

			ynet.SendResponse(ysendConn, ycomm.ResponseInfo{Ok: false, Message: "大海模式还未实现", Status: "no"})
		}
	} else {
		//如果请求的对象不存在，告诉ysend 重新更换名称
		ylog.Logf(">>>>ysend发送数据有误[name不存在于yroute]>>>>")
		fmt.Println(">>>>ysend发送数据有误[name不存在于yroute]>>>>")
		ynet.SendResponse(ysendConn, ycomm.ResponseInfo{Ok: false, Message: "name不存在于yroute", Status: "no"})

		ysendConn.Close()
	}

	//1.收到ysend的单文件传输请求时,解析name字段,然后根据name字段找到相应的yrecv信息
	// 找到后, 判断是否可以直连。
	//
	//2.如果可以直连(溪流)
	//  2.1 与yrecv直接发起TCP连接.
	//	   2.1.1 如果连接成功,发出YROUTE_SEND_SINGLE_FILE(y_s_s_f)命令.
	//	   2.1.2 发送文件名称大小的信息,等待yrecv的响应,yrecv收到后响应Response{Ok: true}
	//     2.1.3 回复ysend, 表示可以准备发送文件
	//     2.1.4 ysend收到回复后,立即传输文件流数据
	//     2.1.5 yroute开始转发
	//  2.2 如果没有成功发起TCP连接
	//     2.2.1 返回失败信息给ysend
	//     2.2.2 ysend 收到失败信息后立即断开与yroute的连接
	//     2.2.3 ysend 退出发送功能
	//
	//3.如果不可用直连(大海),\
	//
	//
	//
	//向其发送 YROUTE_SEND_SINGLE_FILE(y_s_s_f)命令。

	//2.等待yrecv的接收响应, 接收完毕后。 开始发送文件名称信息. 然后等待响应
	//
	//3.收到响应后,开始同步文件数据
}

func doRouter(ip string) {
	bindNet := ynet.GetBindNet(ip, bindPort)

	if bindNet != nil {
		ylog.Logf("Router启动成功于[" + ip + ":" + bindPort + "]")
		for {
			accept, err := bindNet.Accept()
			if err != nil {
				fmt.Println("错误(ERROR): accept错误=>[", err, "]")
				continue
			}

			//获取到请求数据
			requestInfo := ycomm.ParseByteToRequestInfo(ycomm.ReadByte(accept))
			ylog.Logf("收到请求数据===>[", requestInfo.ParseToJsonStr(), "]")
			cmd := requestInfo.Cmd

			switch cmd {
			case ycomm.YRECV_INIT:
				{
					ylog.Logf("yrecv_init请求开始处理===>")

					var tmpYrecvRegInfo = YrecvRegInfo{}
					//注册信息 RequestInfo{cmd: "yrecv_init", data: "name:yms ip:192.168.25.88 cpu:8", other:""}
					//注册信息 RequestInfo{cmd: "yrecv_init", data: "YrecvBase{name:"yms", ip:"10.1.1.1", cpu:"8"}", other:""}
					yrecvBaseInfo := ycomm.ParseStrToYrecvBase(requestInfo.Data)
					tmpYrecvRegInfo.YrecvBase = yrecvBaseInfo
					//保存当前基本连接
					tmpYrecvRegInfo.BaseConn = accept

					ylog.Logf("得到yrecv注册数据体[", yrecvBaseInfo.ParseToJsonStr(), "]")
					ylog.Logf("获取到当前yrecv名称为[", yrecvBaseInfo.Name, "]>>>>")
					if _, ok := yrecvRegInfo[tmpYrecvRegInfo.Name]; ok {
						//更新
						ylog.Logf("delete注册信息>>>>", yrecvRegInfo[tmpYrecvRegInfo.Name])
						delete(yrecvRegInfo, tmpYrecvRegInfo.Name)
						//计数减一
						YrecvRegInfoNum--
					}
					YrecvRegInfoNum++

					yrecvRegInfo[tmpYrecvRegInfo.Name] = tmpYrecvRegInfo

					//发送请求处理完毕信息
					res := ycomm.ResponseInfo{Ok: true, Message: "ok 收到注册信息", Status: "ok"}
					ynet.SendResponse(accept, res)

					//连通性检查
					go checkCanConnect(tmpYrecvRegInfo.Name)

					//处理事件
					//go doYrecvEvent(accept)

				}
			case ycomm.YDECT_MSG:
				{
					ylog.Logf("ydect_msg请求开始处理====>")
					regList := make([]ycomm.YrecvBase, YrecvRegInfoNum)
					i := 0
					for _, v := range yrecvRegInfo {
						regList[i] = v.YrecvBase
						i++
					}
					//获取字符串数据
					yrecvJsonStr := ycomm.ParseYrecvBaseListToJsonStr(regList)
					ylog.Logf("获取注册信息>>>>", yrecvJsonStr)

					res := ycomm.ResponseInfo{
						Ok:      true,
						Message: yrecvJsonStr,
						Status:  "ok",
					}

					ynet.SendResponse(accept, res)

					ylog.Logf("发送完毕>>>>结束")
					//accept.Close()
				}
			case ycomm.YROUTE_SEND_SINGLE_FILE:
				{
					ylog.Logf("YROUTE_SEND_SINGLE_FILE>>>开始处理")
					go doSingleFileHandler(requestInfo, accept)
				}

			}

			//处理连接请求
			//doHandlerRequst(accept)

		}

	} else {
		//退出
		//routeExit()
	}

}

func main() {

	flag.StringVar(&flags.ListenIP, "b", "", "指定监听ip")
	flag.BoolVar(&ycomm.Debug, "debug", true, "debug mode")
	flag.Parse()

	ylog.Logf("输入参数==>[", flags, "]")

	if flags.ListenIP != "" {
		ylog.Logf(">>>启动Router")
		doRouter(flags.ListenIP)
	} else {
		getUsage()
	}

	//<-yRouteExit //阻塞
	fmt.Println("YRouteExit....")
}
