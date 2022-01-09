package main

import (
	"YTools/ycomm"
	"YTools/ylog"
	"YTools/ynet"
	"flag"
	"fmt"
	"net"
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

func (yrr *YrecvRegInfo) parseMap(dataMap map[string]string) {
	for k, v1 := range dataMap {
		switch k {
		case "name":
			yrr.Name = v1
		case "ip":
			yrr.Ip = v1
		case "cpu":
			yrr.Cpu = v1

		}
	}
}

var yrecvRegInfo = make(map[string]YrecvRegInfo)

func doHandlerRequst(accept net.Conn) {

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

					go doYrecvEvent(accept)

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

			}

			//处理连接请求
			doHandlerRequst(accept)

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
