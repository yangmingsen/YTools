package main

import (
	ycomm "YTools/ycomm"
	"YTools/ylog"
	"YTools/ynet"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

//远程ip
var remoteIp string

//多文件远程端口 9949
var multiRemotePort string

//单文件远程端口 8848
var singleRemotePort string

//并发数(用于多文件传输时，默认并发数为cpu的核数
var goroutineNum int

//总文件数
var totalFileNum int32

//目前完成传输文件数
var finishFileNum int32

//传输任务队列
var taskNum chan ycomm.CFileInfo

const (
	SizeB  int64 = ycomm.SizeB
	SizeKB int64 = ycomm.SizeKB
	SizeMB int64 = ycomm.SizeMB
	SizeGB int64 = ycomm.SizeGB
	B            = ycomm.B
	KB           = ycomm.KB
	MB           = ycomm.MB
	GB           = ycomm.GB
)

var a, b float64
var sizeChoose int     //显示条选项
var sendFileSize int64 //发送文件大小(B)

var wg sync.WaitGroup

//初始化
func init() {
	multiRemotePort = ycomm.MultiRemotePort
	singleRemotePort = ycomm.SingleRemotePort
	//cpu的核数
	goroutineNum = runtime.NumCPU()
	totalFileNum = 0
	finishFileNum = 0
}

func sendProgressBar() {
	defer wg.Done()

	var c float64
	c = a
	for {
		d := a - c //获取上次 c-a之间的长度
		c = a

		if d < float64(SizeB) {
			fmt.Printf("\r平均(%.2fB/s), ", d)
		} else if d < float64(SizeKB) {
			fmt.Printf("\r平均(%.2fKB/s), ", d/1024)
		} else if d < float64(SizeMB) {
			fmt.Printf("\r平均(%.2fMB/s), ", d/1024/1024)
		}

		switch sizeChoose {
		case B:
			fmt.Printf("完成度 :%.2f%%, %.0fB => %.0fB", (a/b)*100, a, b)
		case KB:
			fmt.Printf("完成度:%.2f%%, %.2fKB => %.2fKB", (a/b)*100, a/1024, b/1024)
		case MB:
			fmt.Printf("完成度:%.2f%%, %.2fMB => %.2fMB", (a/b)*100, a/1024/1024, b/1024/1024)
		case GB:
			fmt.Printf("完成度:%.2f%%, %.2fGB => %.2fGB", (a/b)*100, a/1024/1024/1024, b/1024/1024/1024)
		}

		if a >= b {
			fmt.Println()
			break
		}

		time.Sleep(time.Second)
	}

}

func sendFile(conn net.Conn, filePath string) {

	file, err0 := os.Open(filePath)
	fStat, _ := file.Stat()

	go sendProgressBar()

	if err0 != nil {
		fmt.Println("os.Open err0:", err0)
		return
	}
	defer file.Close()

	var nowSize int64
	nowSize = 0 //当前下载位置

	buf := make([]byte, 4096)

	//test

	for {
		n, err1 := file.Read(buf)

		nowSize += int64(n)
		a = float64(nowSize) //更新a值

		_, err2 := conn.Write(buf[:n])

		if err1 != nil {
			if err1 == io.EOF {
				fmt.Println("send file ok! FileSize=", fStat.Size(), "   NowSize=", nowSize)
			} else {
				fmt.Println("file.Read err1:", err1)
			}

			break
		}

		if err2 != nil {
			fmt.Println("conn.Write err2:", err2)
			break
		}
	}

}

// filePath 文件路径 格式为 some.zip or some1.txt
//targetIp 远程ip 格式为 ip:port => 192.168.25.72:8848
func parseSingleFileInfo(filePath string, targetIp string) {
	//提取文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		fmt.Println("os.Stat err:", err)
		return
	}
	fileName := fileInfo.Name()    //获取文件名字
	sendFileSize = fileInfo.Size() //获取文件大小

	//设置a,b初始化值
	b = float64(sendFileSize)
	a = float64(0)

	//显示条选择
	if sendFileSize < SizeB {
		sizeChoose = B
	} else if sendFileSize < SizeKB {
		sizeChoose = KB
	} else if sendFileSize < SizeMB {
		sizeChoose = MB
	} else if sendFileSize < SizeGB {
		sizeChoose = GB
	}

	//向服务器发起请求
	conn, err1 := net.Dial("tcp", targetIp)
	if err1 != nil {
		fmt.Println("net.Dial err: ", err)
		return
	}
	defer conn.Close()

	//向目标服务器发送文件名
	_, err2 := conn.Write([]byte(fileName + "+" + strconv.FormatInt(int64(sendFileSize), 10)))
	if err2 != nil {
		fmt.Println("conn.Write err:", err)
		return
	}

	//提示用户发送的文件正在等待对方同意接受
	fmt.Println("您发送的文件正在等待对方接受,请您稍等....")

	//recv remote server 'ok' str.
	buf := make([]byte, 16)
	n, err3 := conn.Read(buf)
	if err3 != nil {
		fmt.Println("conn.Read err:", err3)
		return
	}

	if "yes" == string(buf[:n]) {
		fmt.Println("对方同意接受您的文件,正在发送中....")
		wg.Add(1) //一个 go 等待
		sendFile(conn, filePath)
	} else {
		fmt.Println("对方拒绝了您的发送文件")
	}
}

//获取文件列表
func getFileList(dirpath string) ([]ycomm.CFileInfo, error) {
	var fileList []ycomm.CFileInfo
	dirErr := filepath.Walk(dirpath,
		func(path string, f os.FileInfo, err error) error {
			if f == nil {
				return err
			}

			var fileInfo = ycomm.CFileInfo{Name: f.Name(), IsDir: false, Size: f.Size(), Path: path}
			if f.IsDir() {
				fileInfo.IsDir = true
			}
			fileList = append(fileList, fileInfo)
			return nil
		})
	return fileList, dirErr
}

//同步文件夹
func syncDir(dirList []ycomm.CFileInfo) {

	dSize := len(dirList)
	var sendMsgStr = ""
	for i := 0; i < dSize; i++ {
		if i+1 == dSize {
			sendMsgStr += dirList[i].Path
		} else {
			sendMsgStr += dirList[i].Path + "\n"
		}
	}

	//向服务器发起请求
	connectIP := (remoteIp + ":" + multiRemotePort)
	conn, err1 := net.Dial("tcp", connectIP)
	if err1 != nil {
		fmt.Println("远程服务连接【", connectIP, "】失败")
		panic(err1)
		os.Exit(-1)
	}

	ycomm.WriteMsg(conn, "d")

	ok := ycomm.WriteMsg(conn, sendMsgStr)
	if ok == false {
		fmt.Println("发送目录数据失败【退出】")
		panic(err1)
		os.Exit(-2)
	}

	response := ycomm.ParseStrToResponseInfo(ycomm.ReadMsg(conn))
	fmt.Println("收到对方服务响应：【" + response.Message + "】")

	for {
		response = ycomm.ParseStrToResponseInfo(ycomm.ReadMsg(conn))
		if response.Ok == true {
			break
		} else {
			fmt.Println("对方无法建立目录数据【" + response.Message + "】")
		}
	}
	fmt.Println("同步目录数据结束...")

	defer conn.Close()
}

//显示文件传输进度条
func showSyncFileBar() {
	defer wg.Done()

	nowTime := time.Now()
	var last = int32(0)
	var total = totalFileNum
	for {
		var bar = ""
		current := finishFileNum
		tNum := current - last
		last = current

		percent := (float32(current) / float32(total)) * float32(100)
		for i := 0; i < int(percent)/2; i++ {
			bar += "#"
		}
		fmt.Printf("\r总进度[%-50s] => %.2f%% => %d个文件/s", bar, percent, tNum)
		if current >= total {
			fmt.Println()
			fmt.Println("同步文件完毕...文件数【", total, "】")
			fmt.Println("耗时:%v", time.Since(nowTime))
			break
		}

		time.Sleep(1 * time.Second)
	}
}

//同步文件
func syncFile(fileList []ycomm.CFileInfo) {
	finishFileNum = 0
	totalFileNum = int32(len(fileList))
	taskNum = make(chan ycomm.CFileInfo, totalFileNum)

	//任务输送
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, file := range fileList {
			taskNum <- file
		}
	}()

	// show 进度条
	wg.Add(1)
	go func() {
		showSyncFileBar()
	}()

	//多任务启动
	wg.Add(goroutineNum)
	for i := 0; i < goroutineNum; i++ {
		go func() {
			defer wg.Done()

			//向服务器发起请求
			connectIP := (remoteIp + ":" + multiRemotePort)
			conn, err1 := net.Dial("tcp", connectIP)
			if err1 != nil {
				fmt.Println("远程服务[", connectIP, "]连接失败")
				panic(err1)
				os.Exit(-1)
			}

			//发送传输请求
			ycomm.WriteMsg(conn, "f")

			for {
				tLen := len(taskNum)
				if tLen == 0 { //这意味着没有文件可以传输了
					//fmt.Println("======文件传输结束======")
					ycomm.WriteMsg(conn, "c")
					break
				}

				//获取传输任务
				task, _ := <-taskNum

				//发送开始传输请求【s】标志
				ycomm.WriteMsg(conn, "s")

				//发送文件信息 fileInfo
				ycomm.WriteMsg(conn, ycomm.ParseCFileInfoToJsonStr(task))

				//读取服务响应 检查信息
				response := ycomm.ParseStrToResponseInfo(ycomm.ReadMsg(conn))
				if response.Status == "Exist" { //存在 不传
					//任务完成+1
					atomic.AddInt32(&finishFileNum, 1)
					fmt.Println(response.Message)
					continue
				} else if response.Status == "diffSize" {
					fmt.Println(response.Message)
				}

				//读取接收方 建立文件信息
				response = ycomm.ParseStrToResponseInfo(ycomm.ReadMsg(conn))
				if response.Ok == false { //mean 没有成功
					fmt.Println(response.Message)
					continue
				}

				//fmt.Println("收到服务请求传输文件响应：" + rcvMsg + " 开始传输文件【" + task.Path + "】")

				if task.Size != 0 {
					//传输文件
					file, err0 := os.Open(task.Path)
					if err0 != nil {
						fmt.Println("文件【" + task.Path + "】打开失败，不传输")
						panic(err0)
						break
					}

					buf := make([]byte, 4096)
					current := int64(0)
					for {
						rLen, err1 := file.Read(buf)
						current += int64(rLen)

						_, err2 := conn.Write(buf[:rLen])

						if err1 != nil {
							if err1 == io.EOF {
								break
							} else {
								fmt.Println("读取本地文件数据失败:", err1)
							}
						}

						if err2 != nil {
							fmt.Println("向服务器发送文件数据错误:", err2)
							break
						}

						if current == task.Size {
							break
						}

					}

					//关闭文件
					file.Close()
				}

				//任务完成+1
				atomic.AddInt32(&finishFileNum, 1)

				//接收服务响应完成
				response = ycomm.ParseStrToResponseInfo(ycomm.ReadMsg(conn))
				//fmt.Println("接收到Server传输文件" + task.Path + "完成响应：" + rcvMsg)
				//fmt.Println()

			}
			//关闭网络流
			conn.Close()

		}()
	}

}

//做多文件传输
func doSendMutliFile(path string) {
	nowTime := time.Now()
	fileList, err := getFileList(path)
	if err != nil {
		panic(err)
		return
	}

	var dirList []ycomm.CFileInfo
	var fList []ycomm.CFileInfo
	for _, info := range fileList {
		if info.IsDir == true {
			dirList = append(dirList, info)
		} else {
			fList = append(fList, info)
		}
	}

	syncDir(dirList)
	fmt.Println("同步目录完毕...目录数【", len(dirList), "】 耗时:%v", time.Since(nowTime))
	syncFile(fList)

	//fmt.Println("同步文件完毕...文件数", len(fList))

}

var flags struct {
	RouteIP        string
	RemoteIP       string
	RemoteName     string
	SingleFilePath string
	MultiFilePath  string
	goNumber       int
}

func getUsage() {
	flag.Usage()
	fmt.Println("例如: ysend -f 文件 -d 目标Ip")
	fmt.Println("例如: ysend -r 文件夹 -d 目标Ip  -sn 并发数")
	fmt.Println("例如: ysend -c 10.55.3.4 -dn yms -f 文件")
	fmt.Println("例如: ysend -c 10.55.3.4 -dn yms -r 文件夹 -sn 并发数")
}

func doSingleFileSend(sendName, filePath, targetIp string) {

	//提取文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		ylog.Logf("文件打开失败>>>>[", err, "]")
		return
	}

	fileName := fileInfo.Name()    //获取文件名字
	sendFileSize = fileInfo.Size() //获取文件大小

	yrouteConn, err0 := ynet.GetRemoteConnection(targetIp, ycomm.RoutePort)
	defer yrouteConn.Close()
	if err0 != nil {
		ylog.Logf("建立到远程route连接失败>>>>[", err0, "]")
		return
	}

	//发送
	var sendMap = make(map[string]string)
	sendMap[ycomm.FILE_NAME] = fileName
	sendMap[ycomm.SEND_TO_NAME] = sendName
	sendMap[ycomm.FILE_SIZE] = strconv.FormatInt(sendFileSize, 10)
	sendMsg := ycomm.ParseMapToStr(sendMap)
	ylog.Logf("发送文件信息数据>>>>[", sendMsg, "]")
	ynet.SendRequest(yrouteConn, ycomm.RequestInfo{Cmd: ycomm.YROUTE_SEND_SINGLE_FILE, Data: sendMsg, Other: "no"})

	//等待yroute响应
	resMsg := ycomm.ReadMsg(yrouteConn)
	ylog.Logf("收到yroute的响应数据>>>>", resMsg)
	resp := ycomm.ParseStrToResponseInfo(resMsg)

	if resp.Ok {
		ylog.Logf(">>>>准备同步文件数据流")

		wg.Add(1)
		sendFile(yrouteConn, filePath)

		ylog.Logf(">>>>同步文件数据流结束")

	} else {
		fmt.Println("异常: ", resp.Message)
	}

}

func doRouter() {

}

func main() {

	flag.StringVar(&flags.RouteIP, "c", "", "路由ip")
	flag.StringVar(&flags.MultiFilePath, "r", "", "./文件夹")
	flag.StringVar(&flags.SingleFilePath, "f", "", "./文件")
	flag.StringVar(&flags.RemoteIP, "d", "", "目标ip")
	flag.StringVar(&flags.RemoteName, "dn", "", "dn[dest Name]远程目标名字")
	flag.IntVar(&flags.goNumber, "sn", runtime.NumCPU()*2, "并发数[默认cpu*2]")
	flag.BoolVar(&ycomm.Debug, "debug", true, "debug mode")
	flag.Parse()

	ylog.Logf("输入参数==>", flags)

	//直连与自动直连方式选择
	if flags.RouteIP != "" { //自动直连方式
		ylog.Logf("自动直连选择=======>")
		if flags.RemoteName == "" {
			fmt.Println("必须指定 -dn 目标名称")
		}

		if flags.SingleFilePath != "" {
			ylog.Logf("发送单文件开始=======>")
			doSingleFileSend(flags.RemoteName, flags.SingleFilePath, flags.RouteIP)
			ylog.Logf("发送单文件=======>结束")
		} else if flags.MultiFilePath != "" { //多文件

		} else {
			fmt.Println("必须指定传输文件名或文件夹")
		}

	} else if flags.RemoteIP != "" { //直连方式
		ylog.Logf("直连选择=======>")
		//发送单文件还是多文件
		if flags.MultiFilePath != "" {
			//如果是多文件
			sendPath := flags.MultiFilePath
			remoteIp = flags.RemoteIP
			goroutineNum = flags.goNumber

			ylog.Logf("目标ip==>", remoteIp, "===>发送文件===>", sendPath, "===>goNumber===>", goroutineNum)

			//do send
			doSendMutliFile(sendPath)
		} else if flags.SingleFilePath != "" {
			//单文件
			filePath := flags.SingleFilePath
			targetIp := flags.RemoteIP + ":" + singleRemotePort

			ylog.Logf("目标ip==>", targetIp, "===>发送文件", filePath)

			//do parse
			parseSingleFileInfo(filePath, targetIp)

		} else {
			getUsage()
		}

	} else {
		getUsage()
	}
	wg.Wait()

}
