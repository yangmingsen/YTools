package main

import (
	"YTools/ycomm"
	"YTools/ynet"
	"flag"
	"fmt"
	"io"
	"log"
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

//
var SizeLimit = int64(1 * ycomm.Size1GB)

const (
	SizeB  = ycomm.SizeB
	SizeKB = ycomm.SizeKB
	SizeMB = ycomm.SizeMB
	SizeGB = ycomm.SizeGB
	B      = ycomm.B
	KB     = ycomm.KB
	MB     = ycomm.MB
	GB     = ycomm.GB
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

var logger = log.New(os.Stderr, "", log.Lshortfile|log.LstdFlags)

func yprint(a ...interface{}) {
	//先打印输出
	fmt.Println(a)
}
func logf(f string, v ...interface{}) {
	if ycomm.Debug {
		logger.Output(2, fmt.Sprintf(f, v...))
	}
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

func sendSmallFile(conn net.Conn, filePath string) {
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
func prepareSendSingleFile(filePath string, targetIp string) {
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
	baseConn, err1 := net.Dial("tcp", targetIp)
	if err1 != nil {
		fmt.Println("net.Dial err: ", err1)
		return
	}
	defer baseConn.Close()

	//向目标服务器发送文件名
	initInfo := ycomm.RequestInfo{Cmd: ycomm.FILE_INFO_TIP}.ParseToJsonStr()
	ynet.SendStr(baseConn, initInfo)

	//向目标服务器发送文件名
	fileInfoBytes := []byte(fileName + "+" + strconv.FormatInt(sendFileSize, 10))
	ynet.Send(baseConn, fileInfoBytes)

	//提示用户发送的文件正在等待对方同意接受
	fmt.Println("您发送的文件正在等待对方接受,请您稍等....")
	//recv remote server 'ok' str.
	buf, _ := ynet.Recv(baseConn)
	if "yes" == string(buf) {
		fmt.Println("对方同意接受您的文件,准备发送....")
		if ycomm.IsSmallSend(sendFileSize) {
			sendSmallFile(baseConn, filePath)
		} else {
			//计算分片数及区间
			sliceFun := func(fSzie int64) []ycomm.FileSliceInfo {
				for i := goroutineNum; ; i++ {
					sliceLen := fSzie / int64(i)
					if sliceLen < SizeLimit {
						sliceNum := i
						fsi := make([]ycomm.FileSliceInfo, sliceNum)
						for i := 0; i < sliceNum; i++ {
							if i+1 == sliceNum {
								fsi[i] = ycomm.FileSliceInfo{Id: i, Start: sliceLen * int64(i), Length: fSzie - sliceLen*int64(i)}
							} else {
								fsi[i] = ycomm.FileSliceInfo{Id: i, Start: sliceLen * int64(i), Length: sliceLen}
							}
						}
						return fsi
					}
				}
			}
			sliceList := sliceFun(sendFileSize)
			//总任务数
			totalTaskNum := len(sliceList)
			logf("分片总数：", totalTaskNum)
			//多线程共享channel
			sliceTaskChannel := make(chan ycomm.FileSliceInfo, totalTaskNum)
			for _, v := range sliceList {
				sliceTaskChannel <- v
			}

			//确定双方连接数
			curCPUCoreNum := goroutineNum // runtime.NumCPU()
			logf("当前cpu核心数：", curCPUCoreNum)
			conn1, err1 := ynet.Socket1(targetIp)
			if err1 != nil {
				yprint("确定双方连接数，建立连接时报错: ", err1)
			}
			ynet.Send(conn1, ycomm.RequestInfo{Cmd: ycomm.BIGFILE_INIT}.ParseToByte())

			//Responsenfo{OK: true, Message: "12", Status: "OK"}
			rBytes, _ := ynet.Recv(conn1)
			serverCoreInfo := ycomm.ParseByteToResponseInfo(rBytes)
			conn1.Close()

			logf("收到来自Server的CPU核心数信息: ", string(rBytes))
			serverCPUCoreNum, _ := strconv.Atoi(serverCoreInfo.Message)
			min := func(a, b int) int {
				if a > b {
					return b
				}
				return a
			}

			calculateCoreTaskNum := func(totalTNum, coreCpuNum int) int {
				if totalTNum > coreCpuNum*3 {
					return coreCpuNum * 2
				}

				return totalTNum / 2
			}

			//传输线程数
			coreTaskNum := calculateCoreTaskNum(totalTaskNum, min(curCPUCoreNum, serverCPUCoreNum))
			logf("确定双方建立传输连接数：", coreTaskNum)
			//完成控制
			finishTaskChannel := make(chan bool, 1)
			//任务完成统计channel ,这里的数量应该线程数
			taskFinishNumChannel := make(chan int, coreTaskNum)
			go func() {
				cnt := 0
				for {
					cnt = cnt + <-taskFinishNumChannel
					if cnt == totalTaskNum {
						break
					}
				}
				//所有任务完成通知
				finishTaskChannel <- true
			}()

			sendTargetFile, err3 := os.Open(filePath)
			defer sendTargetFile.Close()
			if err3 != nil {
				yprint("打开发送文件失败. ", err3)
				//目标文件打开失败直接退出
				os.Exit(-1)
			}
			//并发传输
			for i := 0; i < coreTaskNum; i++ {
				id := i
				go func() {
					goroutineName := "goroutine[" + strconv.Itoa(id) + "]"
					curConn, err2 := ynet.Socket1(targetIp)
					if err2 != nil {
						yprint("建立传输分片连接失败. id=", id, " 错误信息:", err2)
						return
					}
					ynet.Send(curConn, ycomm.RequestInfo{Cmd: ycomm.BIGFILE_SLICE_SYNC}.ParseToByte())
					var buf []byte = nil
					for {
						if len(sliceTaskChannel) == 0 {
							break
						}
						//获取一个任务
						task := <-sliceTaskChannel
						if buf == nil {
							buf = make([]byte, task.Length)
						}
						if int64(len(buf)) != task.Length {
							buf = make([]byte, task.Length)
						}

						task.Name = ycomm.GetFileName(sendTargetFile.Name())
						ycomm.FileReadByOffset1(sendTargetFile, buf, task.Start)
						if task.Hash == "" {
							task.Hash = ycomm.Md5Hash(buf)
						}

						jsonStr := task.ParseFileSliceToJsonStr()
						logf(goroutineName, "发送同步头数据数据：", jsonStr)
						ynet.SendStr(curConn, jsonStr)

						//发送流
						curConn.Write(buf)

						//是否需要重新传输该任务
						tmpBytes, _ := ynet.Recv(curConn)
						logf(goroutineName, "收到来自Server是否重传该Slice文件响应:", string(tmpBytes))
						retryInfo := ycomm.ParseByteToResponseInfo(tmpBytes)
						if !retryInfo.Ok {
							//重新加入队列
							sliceTaskChannel <- task
						} else {
							//如果完成
							taskFinishNumChannel <- 1
						}

					}
					curConn.Close()
				}()
			}

			//等待完成
			<-finishTaskChannel

			//发送最后合并指令
			finishConn, err10 := ynet.Socket1(targetIp)
			defer finishConn.Close()
			if err10 != nil {
				yprint("建立finshConn失败: ", err10)
				return
			}
			ynet.Send(finishConn, ycomm.RequestInfo{Cmd: ycomm.BIGFILE_SLICE_SYNC_FINISH}.ParseToByte())
		}
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

func doSyncDir(conn net.Conn, dirList []ycomm.CFileInfo) {
	dSize := len(dirList)
	var sendMsgStr = ""
	for i := 0; i < dSize; i++ {
		if i+1 == dSize {
			sendMsgStr += dirList[i].Path
		} else {
			sendMsgStr += dirList[i].Path + "\n"
		}
	}

	ycomm.WriteMsg(conn, "d")

	ok := ycomm.WriteMsg(conn, sendMsgStr)
	if ok == false {
		fmt.Println("发送目录数据失败【退出】")
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
}

//同步文件夹
func syncDir(dirList []ycomm.CFileInfo) {

	//向服务器发起请求
	connectIP := (remoteIp + ":" + multiRemotePort)
	conn, err1 := net.Dial("tcp", connectIP)
	if err1 != nil {
		fmt.Println("远程服务连接【", connectIP, "】失败")
		panic(err1)
		os.Exit(-1)
	}

	doSyncDir(conn, dirList)

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

	syncFileFlag <- true
}

var syncFileFlag = make(chan bool)

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
			logf("启动任务>>>>", i)
			defer wg.Done()

			var conn net.Conn
			if flags.RouteIP != "" {
				socket, err01 := ynet.Socket(flags.RouteIP, ycomm.RoutePort)
				if err01 != nil {
					logf("创建到yroute连接失败>>>>>", err01)
					//这时需要发送一个请求到yroute，以减少一个连接
				}

				var dMap = make(map[string]string)
				dMap[ycomm.REMOTE_NAME] = flags.RemoteName
				mStr := ycomm.ParseMapToStr(dMap)
				logf("ysend发起文件流请求>>>>>", mStr)

				ynet.SendRequest(socket, ycomm.RequestInfo{
					Cmd:  ycomm.YSEND_MUL_FILE_SYNC2,
					Data: mStr,
				})
				conn = socket
			} else {
				//向服务器发起请求
				connectIP := (remoteIp + ":" + multiRemotePort)
				conn1, err1 := net.Dial("tcp", connectIP)
				if err1 != nil {
					fmt.Println("远程服务[", connectIP, "]连接失败")
					panic(err1)
					os.Exit(-1)
				}
				conn = conn1
			}

			logf(">>>准备发起数据>>>>f")
			//发送传输请求
			ycomm.WriteMsg(conn, "f")
			logf(">>>已发起数据>>>>f")

			for {
				tLen := len(taskNum)
				if tLen == 0 { //这意味着没有文件可以传输了
					//fmt.Println("======文件传输结束======")
					ycomm.WriteMsg(conn, "c")
					break
				}

				//获取传输任务
				task, _ := <-taskNum
				logf("开始传输任务>>>>", task)

				//发送开始传输请求【s】标志
				ycomm.WriteMsg(conn, "s")

				logf("已发送数据标志>>>>s")

				//发送文件信息 fileInfo
				ycomm.WriteMsg(conn, ycomm.ParseCFileInfoToJsonStr(task))
				logf("已发送文件数据>>>>", ycomm.ParseCFileInfoToJsonStr(task))

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
				logf("传输文件", task.Name, "完毕>>>>>>>")
				//接收服务响应完成
				response = ycomm.ParseStrToResponseInfo(ycomm.ReadMsg(conn))
				logf("收到接收到响应>>>>>", ycomm.ParseResponseToJsonStr(response))
				//fmt.Println("接收到Server传输文件" + task.Path + "完成响应：" + rcvMsg)
				//fmt.Println()

			}
			//关闭网络流
			conn.Close()

		}()
	}

}

func getDirAndFileInfo(path string) (dirList, fList []ycomm.CFileInfo) {

	fileList, err := getFileList(path)
	if err != nil {
		panic(err)
		return
	}

	for _, info := range fileList {
		if info.IsDir == true {
			dirList = append(dirList, info)
		} else {
			fList = append(fList, info)
		}
	}

	return dirList, fList
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
		logf("文件打开失败>>>>[", err, "]")
		return
	}

	fileName := fileInfo.Name()    //获取文件名字
	sendFileSize = fileInfo.Size() //获取文件大小

	yrouteConn, err0 := ynet.GetRemoteConnection(targetIp, ycomm.RoutePort)
	defer yrouteConn.Close()
	if err0 != nil {
		logf("建立到远程route连接失败>>>>[", err0, "]")
		return
	}

	//发送
	var sendMap = make(map[string]string)
	sendMap[ycomm.FILE_NAME] = fileName
	sendMap[ycomm.SEND_TO_NAME] = sendName
	sendMap[ycomm.FILE_SIZE] = strconv.FormatInt(sendFileSize, 10)
	sendMsg := ycomm.ParseMapToStr(sendMap)
	logf("发送文件信息数据>>>>[", sendMsg, "]")
	ynet.SendRequest(yrouteConn, ycomm.RequestInfo{Cmd: ycomm.YSEND_TO_YROUTE_TO_YRECV_SINGLE_FILE, Data: sendMsg, Other: "no"})

	//等待yroute响应
	resMsg := ycomm.ReadMsg(yrouteConn)
	logf("收到yroute的响应数据>>>>", resMsg)
	resp := ycomm.ParseStrToResponseInfo(resMsg)

	if resp.Ok {
		logf(">>>>准备同步文件数据流")

		wg.Add(1)
		sendSmallFile(yrouteConn, filePath)

		logf(">>>>同步文件数据流结束")

	} else {
		fmt.Println("异常: ", resp.Message)
	}

}

func doMultiFileSend(sendPath string) {

	routConn, err0 := ynet.Socket(flags.RouteIP, ycomm.RoutePort)
	if err0 != nil {
		logf("与Route建立连接失败>>>>>", err0)
		return
	}

	var dMap = make(map[string]string)
	dMap[ycomm.REMOTE_NAME] = flags.RemoteName
	mStr := ycomm.ParseMapToStr(dMap)

	ynet.SendRequest(routConn, ycomm.RequestInfo{Cmd: ycomm.YSEND_DIR_DATA_SYNC, Data: mStr, Other: "no"})
	defer routConn.Close()
	rcvMsg := ycomm.ReadMsg(routConn)
	logf("收到来自yroute的目录通信信息>>>", rcvMsg)
	reps := ycomm.ParseStrToResponseInfo(rcvMsg)

	if reps.Ok == false {
		logf("同步目录数据错误>>>>>", reps.Message)
		fmt.Println("同步目录数据错误>>>>>", reps.Message)

		return
	}

	logf("收到route>>>>", reps.Message)

	logf("读取目录/文件数据>>>>")
	dirList, fList := getDirAndFileInfo(sendPath)
	logf("读取目录/文件数据>>>>完毕>>>开始发送目录数据")
	doSyncDir(routConn, dirList)
	routConn.Close() //关掉
	logf("完毕>>>发送目录数据>>>ok")

	logf(">>>准备发送文件列表数据")

	routConn2, err2 := ynet.Socket(flags.RouteIP, ycomm.RoutePort)
	if err2 != nil {
		logf("与Route建立连接失败>>>>>", err0)
		return
	}

	ynet.SendRequest(routConn2, ycomm.RequestInfo{
		Cmd:  ycomm.YSEND_MUL_FILE_SYNC,
		Data: mStr,
	})

	logf(">>>准备发送文件列表数据完毕>>>等待接收yroute响应")
	rcvMsg2 := ycomm.ReadMsg(routConn2)
	logf("接收到yroute响应>>>>>", rcvMsg2)
	resp := ycomm.ParseStrToResponseInfo(rcvMsg2)
	routConn2.Close()

	logf("接收到yroute响应消息>>>>>", resp.Message)
	if resp.Ok {
		logf("准备同步文件流>>>>>")
		syncFile(fList)
		<-syncFileFlag
		logf("准备同步文件流>>>>>结束")
	} else {
		logf("同步文件流失败>>>>>")
	}

}

func main() {

	flag.StringVar(&flags.RouteIP, "c", "", "路由ip")
	flag.StringVar(&flags.MultiFilePath, "r", "", "./文件夹")
	flag.StringVar(&flags.SingleFilePath, "f", "", "./文件")
	flag.StringVar(&flags.RemoteIP, "d", "", "目标ip")
	flag.StringVar(&flags.RemoteName, "dn", "", "dn[dest Name]远程目标名字")
	flag.IntVar(&flags.goNumber, "sn", runtime.NumCPU()*2, "并发数[默认cpu*2]")
	flag.BoolVar(&ycomm.Debug, "debug", false, "debug mode")
	flag.Parse()

	//flags.RemoteIP = "127.0.0.1"
	//ycomm.Debug = true
	//flags.SingleFilePath = "G:\\2023_Data\\resume\\online-杨铭森-v202309.pdf"

	logf("输入参数==>", flags)

	//直连与自动直连方式选择
	if flags.RouteIP != "" { //自动直连方式
		logf("自动直连选择=======>")
		if flags.RemoteName == "" {
			fmt.Println("必须指定 -dn 目标名称")
		}

		if flags.SingleFilePath != "" {
			logf("发送单文件开始=======>")
			doSingleFileSend(flags.RemoteName, flags.SingleFilePath, flags.RouteIP)
			logf("发送单文件=======>结束")
		} else if flags.MultiFilePath != "" { //多文件
			regList := ynet.GetRemoteRegInfo(flags.RouteIP)
			var exsit = false
			for _, reg := range regList {
				if reg.Name == flags.RemoteName {
					exsit = true
					break
				}
			}

			if exsit == false {
				logf("不存在目标名称>>>", flags.RemoteName, ">>>>请检查")
				return
			}

			goroutineNum = ycomm.GOBAL_TASK_NUM
			logf("发送多文件>>>>>开始")
			doMultiFileSend(flags.MultiFilePath)
			logf("发送多文件>>>>>结束")

		} else {
			fmt.Println("必须指定传输文件名或文件夹")
		}

	} else if flags.RemoteIP != "" { //直连方式
		logf("直连选择=======>")
		//发送单文件还是多文件
		if flags.MultiFilePath != "" {
			//如果是多文件
			sendPath := flags.MultiFilePath
			remoteIp = flags.RemoteIP
			goroutineNum = flags.goNumber

			logf("目标ip==>", remoteIp, "===>发送文件===>", sendPath, "===>goNumber===>", goroutineNum)

			//do send
			doSendMutliFile(sendPath)
		} else if flags.SingleFilePath != "" {
			//单文件
			filePath := flags.SingleFilePath
			targetIp := flags.RemoteIP + ":" + singleRemotePort

			logf("目标ip==>", targetIp, "===>发送文件", filePath)

			//do parse
			prepareSendSingleFile(filePath, targetIp)

		} else {
			getUsage()
		}

	} else {
		getUsage()
	}
	wg.Wait()

}
