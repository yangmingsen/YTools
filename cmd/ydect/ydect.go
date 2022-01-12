package main

import (
	ycomm "YTools/ycomm"
	"YTools/ylog"
	"YTools/ynet"
	"flag"
	"fmt"
	"net"
	"os"
	"sync"
)

var wg sync.WaitGroup

func doDetect(detectCondition string, listenIP string) {
	nr2 := ycomm.ParseUdpFormat(detectCondition)
	for x := 2; x < 255; x++ {
		var host = byte(x)
		socket, err1 := net.DialUDP("udp", nil, &net.UDPAddr{
			IP:   net.IPv4(nr2[0], nr2[1], nr2[2], host),
			Port: 8849, //对应yrecv
		})
		if err1 != nil {
			fmt.Println("连接探测服务端[", nr2, "]失败，err:", err1)
			continue
		}
		defer socket.Close()
		_, err2 := socket.Write([]byte(listenIP)) //发送主机信息

		if err2 != nil {
			fmt.Println("发送探测数据失败，err:", err2)
		}
	}

}

var flags struct {
	ListenIP string
	RouteIP  string
	DetectIP string
}

func getUsage() {
	flag.Usage()
	fmt.Println("例如: ydect -b 10.3.4.2")
	fmt.Println("例如: ydect -c 150.33.44.23")
}

func doRouteDect(detectIp string) {
	list := ynet.GetRemoteRegInfo(detectIp)

	//遍历响应数据
	fmt.Println("Route注册信息:")
	for _, v := range list {
		v.Show()
	}

}

func main() {

	flag.StringVar(&flags.RouteIP, "c", "", "路由ip")
	flag.StringVar(&flags.ListenIP, "b", "", "指定监听ip")
	flag.StringVar(&flags.DetectIP, "d", "", "指定探测ip")
	flag.BoolVar(&ycomm.Debug, "debug", true, "debug mode")
	flag.Parse()

	ylog.Logf("输入参数==>[", os.Args, "]")

	if flags.RouteIP != "" {
		//do routeDect
		doRouteDect(flags.RouteIP)
	} else if flags.DetectIP != "" {
		if flags.ListenIP == "" {
			ylog.Logf("错误(ERROR): ListenIP为空>>>>>退出")
			os.Exit(01)
		}

		doDect01(flags.ListenIP, flags.DetectIP)
	}

}

func doDect01(listenIp, detectIp string) {
	nr := ycomm.ParseUdpFormat(listenIp)

	listen, err0 := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(nr[0], nr[1], nr[2], nr[3]),
		Port: 8850, //port + 1 => 8849
	})
	if err0 != nil {
		fmt.Println("UDP建立失败")
		panic(err0)
		os.Exit(-1)
	}
	fmt.Println("detect recv server running in " + listenIp)

	defer listen.Close()

	//do udp fun
	wg.Add(1)
	//接收文件
	go func() {
		defer wg.Done()
		doDetect(detectIp, listenIp) //detectip
	}()

	for {
		var data [64]byte
		n, _, err := listen.ReadFromUDP(data[:]) // 接收数据
		if err != nil {
			fmt.Println("read udp failed, err:", err)
			continue
		}
		fmt.Println("find: " + string(data[:n]))

	}

	wg.Wait()
}
