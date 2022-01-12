package main

import (
	"YTools/ycomm"
	"YTools/ylog"
	"YTools/ynet"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
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
	MultiConn2 chan net.Conn
}

var yrecvRegInfo = make(map[string]YrecvRegInfo)

//同步器 name->value
var syncFlag = make(map[string]chan bool)

//同步文件数
var syncFilesFlagNum = ycomm.GOBAL_TASK_NUM
var syncFilesFlag = make(map[string]chan net.Conn)

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

func doTransferStream(fileSize int64, from, to net.Conn) bool {
	//开始转发流数据
	buf := make([]byte, 4096)
	var cur int64 = 0
	for {
		rn, err0 := from.Read(buf)
		_, err1 := to.Write(buf[:rn])
		cur += int64(rn)

		if cur == fileSize {
			ylog.Logf("传输完毕>>>>>")
			return true
		}
		if err0 != nil && err0 != io.EOF {
			ylog.Logf("读取网络流出错>>>>[", err0, "]")
			return false
		}
		if err1 != nil {
			ylog.Logf("写入网络流出错>>>>[", err0, "]")
			return false
		}

	}
}

func doSingleFileSync(fileSize int64, dMap map[string]string, from, to net.Conn) {
	//向yrecv发送命令
	ycomm.WriteMsg(to, ycomm.YSEND_TO_YROUTE_TO_YRECV_SINGLE_FILE)

	//发送文件大小信息给yrecv
	var sendMap = make(map[string]string)
	sendMap["fileName"] = dMap["fileName"]
	sendMap["fileSize"] = dMap["fileSize"]
	sendStr := ycomm.ParseMapToStr(sendMap)
	ylog.Logf("发送文件大小信息>>>>", sendStr)
	ycomm.WriteMsg(to, sendStr)

	//等待yrecv的响应
	rcvMsg := ycomm.ReadMsg(to)
	ylog.Logf("收到yrecv响应>>>>", rcvMsg)
	resp := ycomm.ParseStrToResponseInfo(rcvMsg)
	if resp.Ok {
		//发送ok给ysend
		ynet.SendResponse(from, ycomm.ResponseInfo{Ok: true, Message: "ok prepare fileSteam", Status: "ok"})
		ylog.Logf("准备完毕...开始. ysend=>yroute=>yrecv")

		doTransferStream(fileSize, from, to)
	}
}

//处理单文件路由传输
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
//	 3.1 大海模式传输每次必须,重新建立连接.
//	    3.1.1 使用BaseConn发送请求建立连接命令 YRECV_BASECONN_SINGLE,
//		3.1.2 然后进入等待BaseConn的响应(比如yrecv建立到yroute的连接是否成功) Response(ok: true/false)
//      3.1.3 如果收到true则开始信号量等待
//   3.2 yrecv 收到 YRECV_BASECONN_SINGLE命令后, 立即主动向yroute发起新连接请求 YRECV_REQUEST_ESTABLISH_CONN 命令
//   3.3 yroute收到请求后,立即将新singleConn连接,存入yrecvReg中. 立即通知大海模式线程,
//   3.4 大海模式收到信号量后,判断信号量状态
//		3.4.1 如果为true
//			3.4.1.1 向 yrecv 发出YROUTE_SEND_SINGLE_FILE(y_s_s_f)命令.
//			3.4.1.2  发送文件名称大小的信息,等待yrecv的响应,yrecv收到后响应Response{Ok: true}
//			3.4.1.3  回复ysend, 表示可以准备发送文件
//			3.4.1.4 ysend收到回复后,立即传输文件流数据
//		3.4.2 如果为false
//			3.4.2.1 那么响应ysend, 文件不可发送(超时未收到yrecv的响应)
//
//
//
//向其发送 YSEND_TO_YROUTE_TO_YRECV_SINGLE_FILE(y_s_s_f)命令。
//
//2.等待yrecv的接收响应, 接收完毕后。 开始发送文件名称信息. 然后等待响应
//
//3.收到响应后,开始同步文件数据
func doSingleFileHandler(requestInfo ycomm.RequestInfo, ysendConn net.Conn) {
	defer ysendConn.Close()

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
	}
	ylog.Logf("的body数据>>>>>存在", ycomm.FILE_SIZE)

	if _, ok := dMap[ycomm.FILE_NAME]; ok == false {
		ylog.Logf(">>>>ysend发送数据有误fileName不存在]>>>>")
		fmt.Println(">>>>ysend发送数据有误[fileName不存在]>>>>")

		ynet.SendResponse(ysendConn, ycomm.ResponseInfo{Ok: false, Message: "fileName不存在", Status: "no"})
	}
	ylog.Logf("的body数据>>>>>存在", ycomm.FILE_NAME)

	if regInfo, ok := yrecvRegInfo[cName]; ok {
		ylog.Logf("已匹配>>>", yrecvRegInfo[cName], ">>>>准备连接")
		if regInfo.CanConn { //溪流模式
			//if false { //溪流模式
			ylog.Logf("溪流模式开始连接yrecv>>>>>", regInfo.Ip, ">>>>")
			yrecvConn, err0 := ynet.GetRemoteConnection(regInfo.Ip, ycomm.MultiRemotePort)
			defer yrecvConn.Close()
			if err0 == nil { //如果连接成功
				doSingleFileSync(fileSize, dMap, ysendConn, yrecvConn)
				ylog.Logf("溪流模式传输结束>>>>>")

			} else { //如果连接yrecv失败
				ylog.Logf("开始连接yrecv>>>>>", regInfo.Ip, ">>>>")

				ynet.SendResponse(ysendConn, ycomm.ResponseInfo{Ok: false, Message: "connect yrecv failed", Status: "no"})
			}
		} else { //大海模式
			ylog.Logf("启动大海模式>>>>>")
			//获取BaseConn
			ynet.SendRequest(regInfo.BaseConn, ycomm.RequestInfo{
				Cmd:   ycomm.YRECV_BASECONN_SINGLE,
				Data:  "no",
				Other: "no",
			})
			ylog.Logf("发送完毕>>>>等待接收响应>>>>")

			rcvMsg := ycomm.ReadMsg(regInfo.BaseConn)
			ylog.Logf("接收到响应数据>>>>", rcvMsg)
			resp := ycomm.ParseStrToResponseInfo(rcvMsg)
			if resp.Ok {
				//如果成功
				ylog.Logf("等待信号量响应>>>>")
				var ok = <-syncFlag[cName] //阻塞
				ylog.Logf("信号量响应>>>>", ok)

				//获取yrecv端连接
				var to = yrecvRegInfo[cName].SingleConn

				if ok {
					ylog.Logf("准备开始传输文件>>>>")
					doSingleFileSync(fileSize, dMap, ysendConn, to)
					ylog.Logf("传输文件结束>>>>")
				}

			} else {
				//如果失败
				ylog.Logf("失败>>>>", resp.Message)
				ynet.SendResponse(ysendConn, ycomm.ResponseInfo{Ok: false, Message: resp.Message, Status: "no"})
			}

		}
	} else {
		//如果请求的对象不存在，告诉ysend 重新更换名称
		ylog.Logf(">>>>ysend发送数据有误[name不存在于yroute]>>>>")
		ynet.SendResponse(ysendConn, ycomm.ResponseInfo{Ok: false, Message: "name不存在于yroute", Status: "no"})
	}

}

func doRouter(ip string) {
	bindNet := ynet.GetBindNet(ip, bindPort)

	if bindNet != nil {
		ylog.Logf("Router启动成功于[" + ip + ":" + bindPort + "]")
		for {
			accept, err := bindNet.Accept()
			if err != nil {
				ylog.Logf("错误(ERROR): accept错误=>[", err, "]")
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
					syncFlag[tmpYrecvRegInfo.Name] = make(chan bool)

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
			case ycomm.YSEND_TO_YROUTE_TO_YRECV_SINGLE_FILE:
				{
					ylog.Logf("YSEND_TO_YROUTE_TO_YRECV_SINGLE_FILE>>>开始处理")
					go doSingleFileHandler(requestInfo, accept)
				}
			case ycomm.YRECV_REQUEST_ESTABLISH_CONN:
				{
					ylog.Logf("处理>>>>YRECV_REQUEST_ESTABLISH_CONN")
					dMap := ycomm.ParseStrToMapData(requestInfo.Data)
					tt := dMap[ycomm.TO_TYPTE]
					if tt == ycomm.SINGLE { //
						cName := dMap[ycomm.HOSTNAME]
						if reg, ok := yrecvRegInfo[cName]; ok {
							ylog.Logf(">>>>>存在>>>开始更新SingleConn")
							reg.SingleConn = accept
							yrecvRegInfo[cName] = reg
							ylog.Logf(">>>>>存在>>>更新SingleConn完毕>>>当前Map>>", yrecvRegInfo[cName])
							ylog.Logf(">>>>>发送信号量>>>true")
							syncFlag[cName] <- true
							ylog.Logf(">>>>>发送信号量>>>true结束")

						} else {
							ylog.Logf(">>>>>不存在>>>", cName)
						}

					} else if tt == ycomm.MULTI {
						ylog.Logf("处理>>>>ycomm.MULTI ")
						cName := dMap[ycomm.HOSTNAME]

						ylog.Logf("准备添加新连接到>>>>", cName, ">>>>conList")
						//连接add到缓冲
						syncFilesFlag[cName] <- accept
						//连接add到缓冲
						connList := syncFilesFlag[cName]
						cLen := len(connList)
						ylog.Logf(">>>当前同步长度>>>", cLen)

						if cLen == syncFilesFlagNum {
							syncFlag[cName] <- true
							ylog.Logf(">>>>已到达指定长度>>>>发送信号量true")
						}

					}
				}
			case ycomm.YSEND_DIR_DATA_SYNC:
				{
					go func() {
						ylog.Logf("处理>>>>YSEND_DIR_DATA_SYNC")
						dMap := ycomm.ParseStrToMapData(requestInfo.Data)

						regInfo := yrecvRegInfo[dMap[ycomm.REMOTE_NAME]]

						ynet.SendRequest(regInfo.BaseConn, ycomm.RequestInfo{
							Cmd:   ycomm.YSEND_DIR_DATA_SYNC,
							Data:  "no",
							Other: "no",
						})

						rcvMsg := ycomm.ReadMsg(regInfo.BaseConn)
						ylog.Logf("收到来自yrecv的创建目录连接结果>>>>", rcvMsg)

						ynet.SendResponse(accept, ycomm.ResponseInfo{
							Ok:      true,
							Message: "ok",
						})

						cNmae := regInfo.Name
						<-syncFlag[cNmae]

						toConn := yrecvRegInfo[cNmae].SingleConn
						ylog.Logf("准备传输目录数据>>>>>")
						//阻塞等待

						ylog.Logf("开始传输目录数据>>>>>")

						ynet.TransferStream(accept, toConn)
					}()

				}
			case ycomm.YSEND_MUL_FILE_SYNC:
				{

					ylog.Logf("处理>>>>YSEND_MUL_FILE_SYNC")
					go doMultiFileSync(accept, requestInfo)

				}
			case ycomm.YSEND_MUL_FILE_SYNC2:
				{

					go func() {
						ylog.Logf("处理>>>>YSEND_MUL_FILE_SYNC2")

						dMap := ycomm.ParseStrToMapData(requestInfo.Data)
						cName := dMap[ycomm.REMOTE_NAME]

						var to = <-syncFilesFlag[cName]
						ylog.Logf("开始转接流>>>>>>")

						ynet.TransferStream(accept, to)
						ylog.Logf("处理>>>>YSEND_MUL_FILE_SYNC2>>>>>>>>")
					}()

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

func doMultiFileSync(conn net.Conn, requestInfo ycomm.RequestInfo) {

	dMap := ycomm.ParseStrToMapData(requestInfo.Data)
	regInfo := yrecvRegInfo[dMap[ycomm.REMOTE_NAME]]

	//用于存储coreNum个yrecv的连接缓冲
	syncFilesFlag[regInfo.Name] = make(chan net.Conn, syncFilesFlagNum)

	dMap[ycomm.CORE_NUM] = strconv.Itoa(syncFilesFlagNum)
	sStr := ycomm.ParseMapToStr(dMap)
	ylog.Logf("发送>>>>YSEND_MUL_FILE_SYNC到yrecv>>>>", sStr)
	ynet.SendRequest(regInfo.BaseConn, ycomm.RequestInfo{
		Cmd:  ycomm.YSEND_MUL_FILE_SYNC,
		Data: sStr,
	})
	ylog.Logf("发送>>>>等待所有连接建立完成")
	rcvMsg := ycomm.ReadMsg(regInfo.BaseConn)
	ylog.Logf("收到来自yrecv的建立连接完毕响应>>>>>", rcvMsg)
	reps := ycomm.ParseStrToResponseInfo(rcvMsg)

	ylog.Logf("yrecv主动建立多文件传输>>>>", reps.Message)
	if reps.Ok {
		var ok = <-syncFlag[regInfo.Name]
		if ok {
			ylog.Logf("收到全部yrecv的主动连接请求")
			ynet.SendResponse(conn, ycomm.ResponseInfo{
				Ok:      true,
				Message: "已全部准备完毕...",
			})

		} else {
			reps.Ok = false
			reps.Message = "超时未收到yrecv的主动连接请求"
			ynet.SendResponse(conn, reps)
		}

	} else {
		ynet.SendResponse(conn, reps)
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
	} else if len(os.Args) == 1 {
		doRouter(ycomm.LOCAL_HOST)
	} else {
		getUsage()
	}

	//<-yRouteExit //阻塞
	fmt.Println("YRouteExit....")
}
