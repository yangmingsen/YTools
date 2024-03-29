package main

import (
	"YTools/ycomm"
	"YTools/ynet"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

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

//多文件监听端口
var mutilFilePort string

//单文件监听
var singleFilePort string

//探测端口
var detectPort int

var wg sync.WaitGroup

func init() {
	mutilFilePort = "9949"
	singleFilePort = "8848"
	detectPort = 8850
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

//传输进度条显示
func showSingleBar(current *int64, total int64) {
	defer wg.Done()

	last := int64(0)
	avgShow := -1

	//显示条选择
	if total < SizeB {
		avgShow = B
	} else if total < SizeKB {
		avgShow = KB
	} else if total < SizeMB {
		avgShow = MB
	} else if total < SizeGB {
		avgShow = GB
	}

	for {
		var bar = ""
		//获取本次长度
		tmpLen := *current - last
		//保存上次结果
		last = *current

		percent := (float64(*current) / float64(total)) * float64(100)
		for i := 0; i < int(percent)/2; i++ {
			bar += "#"
		}
		fmt.Printf("\r总进度[%-50s]", bar)

		a := float64(*current)
		b := float64(total)

		switch avgShow {
		case B:
			fmt.Printf("%.2f%%, %.0fB => %.0fB,", (a/b)*100, a, b)
		case KB:
			fmt.Printf("%.2f%%, %.2fKB => %.2fKB,", (a/b)*100, a/1024, b/1024)
		case MB:
			fmt.Printf("%.2f%%, %.2fMB => %.2fMB,", (a/b)*100, a/1024/1024, b/1024/1024)
		case GB:
			fmt.Printf("%.2f%%, %.2fGB => %.2fGB,", (a/b)*100, a/1024/1024/1024, b/1024/1024/1024)
		}

		d := float64(tmpLen)
		if d < float64(SizeB) {
			fmt.Printf(" 平均(%.2fB/s)", d)
		} else if d < float64(SizeKB) {
			fmt.Printf(" 平均(%.2fKB/s)", d/1024)
		} else if d < float64(SizeMB) {
			fmt.Printf(" 平均(%.2fMB/s)", d/1024/1024)
		}

		if *current >= total {
			fmt.Println("接收完毕")
			break
		}
		time.Sleep(1 * time.Second)

	}

}

//单文件函数
//***********************************************************************
func recvSmallFile(conn net.Conn, fileName string, fileSize int64) {
	defer conn.Close()

	current := int64(0)
	//wg.Add(1)
	go showSingleBar(&current, fileSize)

	file, err0 := os.Create(fileName)
	if err0 != nil {
		fmt.Println("os.Create(fileName) err0:", err0)
		return
	}
	defer file.Close()

	buf := make([]byte, 4096)
	for {
		n, err0 := conn.Read(buf)
		current += int64(n)
		file.Write(buf[:n])

		if err0 != nil && err0 != io.EOF {
			fmt.Println("读取网络流出错>>>>[", err0, "]")
			break
		}

		if current == fileSize {
			//fmt.Println("接收文件 ", fileName, " 完成")
			break
		}

	}

}

// 获取单个ip地址，具有固定性。 容易出错
func getLocalIpv4() string {
	addrs, err := net.InterfaceAddrs() //获取所有ip地址, 包含ipv4,ipv6
	if err != nil {
		panic(err)
	}

	//fmt.Println(addrs)
	for _, addr := range addrs {
		fmt.Println(addr)
	}

	ipv4Addr := addrs[1].String()
	split := strings.Split(ipv4Addr, "/")

	return split[0]
}

//udp fun
func listenUDP(ip string) {

	var listen *net.UDPConn
	var err0 error
	var nr []byte
	ipStr := ip

	if ip != "nil" {
		if strings.Contains(ip, ":") {
			splitStr := strings.Split(ip, ":")
			ipStr = splitStr[0]
		}

		nr = ycomm.ParseUdpFormat(ipStr)

		listen, err0 = net.ListenUDP("udp", &net.UDPAddr{
			IP:   net.IPv4(nr[0], nr[1], nr[2], nr[3]),
			Port: detectPort, //port + 1 => 8849
		})
		if err0 != nil {
			fmt.Println("UDP建立失败【", err0, "】")
			return
		}

	} else {
		listenIp := ycomm.GetLocalIpv4List()
		ipLen := len(listenIp)
		if ipLen == 0 {
			fmt.Println("Not available Ip Address for UdpServer bind...")
			return
		}

		ok := false
		for _, theIp := range listenIp {
			nr = ycomm.ParseUdpFormat(theIp)

			listen, err0 = net.ListenUDP("udp", &net.UDPAddr{
				IP:   net.IPv4(nr[0], nr[1], nr[2], nr[3]),
				Port: detectPort, //port + 1 => 8849
			})
			if err0 != nil {
				fmt.Println("ip: ", theIp, " Can't bind, [", err0, "]")
			} else {
				ipStr = theIp
				ok = true
				break
			}

		}
		if ok == false {
			return
		}

	}

	fmt.Println("DServer Successful running in  " + ipStr + ":8849")

	defer listen.Close()

	//获取hostname
	hostname, _ := os.Hostname()
	var sendInfo = ipStr + "/" + hostname

	for {
		var data [32]byte
		n, addr, err := listen.ReadFromUDP(data[:])
		if err != nil {
			fmt.Println("read udp failed, err:", err)
			continue
		}

		fmt.Printf("Recv detect request data:%v addr:%v len:%v\n", string(data[:n]), addr, n)
		reallyRemoteIP := string(data[:n])
		//recvIpStr := addr.String()
		//sprArr := strings.Split(recvIpStr, ":")
		nr2 := ycomm.ParseUdpFormat(reallyRemoteIP)

		socket, err1 := net.DialUDP("udp", nil, &net.UDPAddr{
			IP:   net.IPv4(nr2[0], nr2[1], nr2[2], nr2[3]),
			Port: 8850, //探测服务器
		})
		if err1 != nil {
			fmt.Println("连接探测服务端[", reallyRemoteIP, "]失败，err:", err)
			continue
		}
		defer socket.Close()
		_, err2 := socket.Write([]byte(sendInfo)) //发送主机信息
		if err2 != nil {
			fmt.Println("响应探测数据失败，err:", err)
			continue
		}

	}

}

const tmpSlice = ycomm.TmpSlice
const SPLIT_FLAG = ycomm.SPLIT_FLAG

func singleFileReceiver(listener net.Listener) {
	//defer wg.Done()
	for {
		//阻塞监听client connection
		conn, err1 := listener.Accept()
		if err1 != nil {
			logf("singleFileReceiver listener.Accept() err1:", err1)
			conn.Close()
		}

		rcvBytes, err0 := ynet.Recv(conn)
		if err0 != nil {
			logf("singleFileReceiver 读取出错: ", err0)
			continue
		}
		rcvMsg := string(rcvBytes)
		logf("收到 ysend 信息>>>>", rcvMsg)
		reqInfo := ycomm.ParseStrToRequestInfo(rcvMsg)

		if reqInfo.Cmd == ycomm.BIGFILE_INIT {
			go func() {
				dirExists := func(path string) bool {
					_, dErr0 := os.Stat(path)
					if dErr0 != nil {
						if os.IsExist(dErr0) {
							return true
						}
						return false
					}
					return true
				}

				cpuNum := runtime.NumCPU()
				isExsit := dirExists(tmpSlice)
				logf("目录是否存在: ", isExsit)
				if !isExsit {
					os.Mkdir(tmpSlice, os.ModePerm)
				}
				ynet.Send(conn, ycomm.ParseResponseToBytes(ycomm.ResponseInfo{
					Ok:      true,
					Message: strconv.Itoa(cpuNum),
					Status:  "Ok",
				}))
			}()
		} else if reqInfo.Cmd == ycomm.BIGFILE_SLICE_SYNC {
			go func() {
				var buf []byte = nil
				for {
					rbs, er1 := ynet.Recv(conn)
					if er1 != nil {
						yprint("bigFileSliceSync收取数据异常：", er1)
						break
					}
					logf("收到bigFileSliceSync数据: ", string(rbs))
					fileSlice := ycomm.ParseByteToFileSlice(rbs)
					if buf == nil {
						buf = make([]byte, fileSlice.Length)
					}
					if int64(len(buf)) != fileSlice.Length {
						buf = make([]byte, fileSlice.Length)
					}
					readStream := func(tConn net.Conn, buf []byte, n int) {
						cnt := 0
						tmpBuf := make([]byte, 1024)
						for {
							rn, tErr := tConn.Read(tmpBuf)
							if tErr != nil {
								yprint("bigFileSliceSync收取流数据时报错：", tErr)
								break
							}
							for i := 0; i < rn; i++ {
								buf[cnt] = tmpBuf[i]
								cnt++
							}
							if cnt >= n {
								break
							}
						}
					}
					readStream(conn, buf, int(fileSlice.Length))
					hash := ycomm.Md5Hash(buf)
					if hash != fileSlice.Hash {
						ynet.Send(conn, ycomm.ParseResponseToBytes(ycomm.ResponseInfo{
							Ok:      false,
							Message: "Hash check error",
							Status:  "Ok",
						}))
						continue
					}
					pathName := tmpSlice + ycomm.GetOsSparator() + fileSlice.Name + SPLIT_FLAG + strconv.Itoa(fileSlice.Id)
					logf("创建文件地址：", pathName)
					//建立文件
					newSliceFile, nErr1 := os.Create(pathName)
					if nErr1 != nil {
						logf("创建新分片文件失败: ", nErr1)
						ynet.Send(conn, ycomm.ParseResponseToBytes(ycomm.ResponseInfo{
							Ok:      false,
							Message: "create new slice file failed",
							Status:  "Ok",
						}))
						continue
					}
					_, nErr2 := newSliceFile.Write(buf[:])
					if nErr2 != nil {
						logf("数据写入新分片文件失败: ", nErr2)
						ynet.Send(conn, ycomm.ParseResponseToBytes(ycomm.ResponseInfo{
							Ok:      false,
							Message: "数据写入新分片文件失败",
							Status:  "Ok",
						}))
						continue
					}
					newSliceFile.Close()

					logf("分片文件：", pathName, " 接收完成")
					ynet.Send(conn, ycomm.ParseResponseToBytes(ycomm.ResponseInfo{
						Ok:      true,
						Message: "success",
						Status:  "Ok",
					}))
				}
				conn.Close()

			}()
		} else if reqInfo.Cmd == ycomm.BIGFILE_SLICE_SYNC_FINISH {
			go func() {
				mergeSliceFiles := func(src string) {
					files, _ := ioutil.ReadDir(src)
					newFileName := ""
					if len(files) > 0 {
						newFileName = strings.Split(files[0].Name(), SPLIT_FLAG)[0]
					}
					logf("newFileName= ", newFileName)
					getId := func(spName string) int {
						idStr := strings.Split(spName, SPLIT_FLAG)[1]
						id, _ := strconv.Atoi(idStr)
						return id
					}

					aFile, err := os.OpenFile(newFileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
					if err != nil {
						fmt.Println("FileSequenceMerge 打开A文件失败:", err)
						return
					}
					defer aFile.Close()
					sortFiles := make([]fs.FileInfo, len(files))
					for i := 0; i < len(files); i++ {
						id := getId(files[i].Name())
						sortFiles[id] = files[i]
					}

					for _, v := range sortFiles {
						appendPath := tmpSlice + ycomm.GetOsSparator() + v.Name()
						bFile, err1 := os.Open(appendPath)
						if err1 != nil {
							logf("打开B文件失败:", err)
							bFile.Close()
							return
						}

						// 从B文件读取内容并追加到A文件
						_, err = io.Copy(aFile, bFile)
						if err != nil {
							logf("追加文件内容失败:", err)
							return
						}

						bFile.Close()
						err1 = os.Remove(appendPath)
						if err1 != nil {
							logf("删除分片文件失败：", err1)
						}
					}

				}
				mergeSliceFiles("./" + tmpSlice)
				logf("合并完成....")

			}()
		} else if reqInfo.Cmd == ycomm.FILE_INFO_TIP {
			go func() {
				buf, _ := ynet.Recv(conn)
				fileInfo := string(buf)
				split := strings.Split(fileInfo, "+")
				fileName := split[0]
				fileSize := split[1]
				//将string转换为64位int
				recvFileSizeInt64, _ := strconv.ParseInt(fileSize, 10, 64)
				addr := conn.RemoteAddr().String() //获得远程ip的格式为 [ip]:[port]
				splitAddr := strings.Split(addr, ":")

				//提示信息
				fmt.Print("您是否（Y/N）愿意接受对方（" + splitAddr[0] + "）向您发送文件: " + fileName + " 大小为: ")
				if recvFileSizeInt64 < SizeB {
					fmt.Printf("%.0fB\n", float64(recvFileSizeInt64))
				} else if recvFileSizeInt64 < SizeKB {
					fmt.Printf("%.2fKB\n", float64(recvFileSizeInt64)/1024)
				} else if recvFileSizeInt64 < SizeMB {
					fmt.Printf("%.2fMB\n", float64(recvFileSizeInt64)/1024/1024)
				} else if recvFileSizeInt64 < SizeGB {
					fmt.Printf("%.2fGB\n", float64(recvFileSizeInt64)/1024/1024/1024)
				}

				//用户选择
				var ch string
				fmt.Scan(&ch)
				ch = strings.ToLower(ch)
				if ch == "y" {
					//同意接受文件
					//conn.Write([]byte("yes"))
					ynet.SendStr(conn, "yes")
					if ycomm.IsSmallSend(recvFileSizeInt64) {
						logf("小文件发送开始...")
						recvSmallFile(conn, fileName, recvFileSizeInt64)
					} else {
						logf("大文件接收开始...")
					}
				} else {
					ynet.SendStr(conn, "no")
				}
			}()
		} else {
			logf("非法请求: ", rcvMsg)
		}

	}

}

//多文件函数
//***********************************************************************

func doCreateServer(port string) net.Listener {
	socket, err := ynet.ServerSocket(port)
	if err != nil {
		logf("建立网络Listen出错>>>>", err)
		return nil
	}
	return socket
}

func doCreateServer2(port string) net.Listener {
	listenIp := ycomm.GetLocalIpv4List()
	ipLen := len(listenIp)
	if ipLen == 0 {
		fmt.Println("Not available Ip Address to bind")
		return nil
	}

	//
	for _, theIp := range listenIp {
		bindServer := ynet.GetBindNet(theIp, port)
		if bindServer != nil {
			return bindServer
		}
	}

	return nil
}

func replaceSeparator(path string) string {
	const separator = os.PathSeparator
	var spa string
	if separator == '\\' {
		spa = "\\"
	} else {
		spa = "/"
	}

	var resPath = ""
	var tmpPath []string
	if strings.Contains(path, "\\") {
		tmpPath = strings.Split(path, "\\")
	} else {
		tmpPath = strings.Split(path, "/")
	}

	for i := 0; i < len(tmpPath); i++ {
		if i+1 == len(tmpPath) {
			resPath += tmpPath[i]
		} else {
			resPath += (tmpPath[i] + spa)
		}
	}

	return resPath
}

// exists returns whether the given file or directory exists or not
func fileExists(path string) (bool, os.FileInfo, error) {
	f, err := os.Stat(path)
	if err == nil {
		return true, f, nil
	}
	if os.IsNotExist(err) {
		return false, nil, nil
	}
	return true, f, err
}

func doDirCreate(conn net.Conn) {
	rcvMsg := ycomm.ReadMsg(conn)
	response := ycomm.ResponseInfo{Ok: true, Message: "收到目录数据", Status: "Ok"}
	ycomm.WriteMsg(conn, ycomm.ParseResponseToJsonStr(response))

	pathArr := strings.Split(rcvMsg, "\n")
	for _, path := range pathArr {
		// os.Mkdir("abc", os.ModePerm)              //创建目录
		// os.MkdirAll("dir1/dir2/dir3", os.ModePerm)   //创建多级目录
		err := os.MkdirAll(replaceSeparator(path), os.ModePerm)
		if err != nil {
			response.Ok = false
			response.Message = err.Error()
			ycomm.WriteMsg(conn, ycomm.ParseResponseToJsonStr(response))
			fmt.Println("创建目录失败【"+path+"】", err)
		}
	}

	response.Ok = true
	response.Message = "目录建立完毕...."
	ycomm.WriteMsg(conn, ycomm.ParseResponseToJsonStr(response))

	//关闭连接
	//defer conn.Close()

	fmt.Println("目录建立完毕....")
}

// 处理目录建立
func doDirHandler(conn net.Conn) {
	defer wg.Done()
	doDirCreate(conn)
}

//
func doFileHandler(conn net.Conn) {
	defer wg.Done()
	//准备断点续传（如果大于25M的文件采取该行为) =>未实现
	//largeSize := int64(26214400)// 25B*1024*1024 => 26214400B =>25MB

	for {
		//接收文件传输标志
		//可能是 s 或 c
		// 如果是c 表示结束传输
		rcvMsg := ycomm.ReadMsg(conn)
		if rcvMsg == "c" {
			fmt.Println("Client传输完毕")
			break
		}
		//计时开始
		nowTime := time.Now()

		//接收文件信息 fileInfo
		fileInfo := ycomm.ParseStrToCFileInfo(ycomm.ReadMsg(conn))

		//转换path
		filePath := replaceSeparator(fileInfo.Path)

		//响应检查信息
		var response = ycomm.ResponseInfo{Ok: true, Message: "", Status: "Ok"}
		exists, finfo, _ := fileExists(filePath)
		if exists { //如果存在
			if finfo.Size() != fileInfo.Size {
				//如果文件大小不一致 也得重传
				err := os.Remove(filePath) //删除错误文件
				if err != nil {
					log.Fatal(err)
				}

				//serverSize := strconv.FormatInt(finfo.Size(), 10)
				//clientSize := strconv.FormatInt(fileInfo.Size, 10)
				response.Message = "接收方文件【" + filePath + "】:" + "大小【" + fileSizeReadable(finfo.Size()) + "】与本地文件【" + fileSizeReadable(fileInfo.Size) + "】大小不一致准备重传. "
				response.Status = "diffSize"

				ycomm.WriteMsg(conn, ycomm.ParseResponseToJsonStr(response))

			} else { //否则 存在不传

				response.Status = "Exist"
				response.Message = "接收方文件【" + filePath + "】存在, 不传."

				ycomm.WriteMsg(conn, ycomm.ParseResponseToJsonStr(response))
				continue
			}

		} else {
			// 可以传送
			ycomm.WriteMsg(conn, ycomm.ParseResponseToJsonStr(response))
		}

		//建立文件
		file, err0 := os.Create(filePath)
		defer file.Close()
		if err0 != nil {
			response.Ok = false
			response.Message = "接收方创建文件【" + filePath + "】失败:" + err0.Error()
			response.Status = "Ok"

			ycomm.WriteMsg(conn, ycomm.ParseResponseToJsonStr(response))
			continue
		} else {
			response.Ok = true
			response.Message = "Ok"
			ycomm.WriteMsg(conn, ycomm.ParseResponseToJsonStr(response))
		}

		if fileInfo.Size != 0 {
			current := int64(0)

			buf := make([]byte, 4096)
			for {
				n, _ := conn.Read(buf)
				current += int64(n)
				file.Write(buf[:n])

				if current >= fileInfo.Size {
					break
				}
			}
		}
		//关闭文件
		file.Close()

		fmt.Println("文件【"+filePath+"】接收完毕，大小【", fileSizeReadable(fileInfo.Size), "】，耗时:【%v】", time.Since(nowTime))

		response.Ok = true
		response.Message = "Ok"
		ycomm.WriteMsg(conn, ycomm.ParseResponseToJsonStr(response))

	}
	conn.Close()

}

//可读化文件大小
func fileSizeReadable(fileSize int64) string {
	var sizeStr string
	//显示条选择
	if fileSize < SizeB {
		sizeStr = strconv.FormatInt(fileSize, 10)
		sizeStr += " B"
	} else if fileSize < SizeKB {
		value := float64(fileSize) / float64(1024)
		sizeStr += fmt.Sprintf("%.2f KB", value)
	} else if fileSize < SizeMB {
		value := float64(fileSize) / float64(1024*1024)
		sizeStr += fmt.Sprintf("%.2f MB", value)
	} else if fileSize < SizeGB {
		value := float64(fileSize) / float64(1024*1024*1024)
		sizeStr += fmt.Sprintf("%.2f GB", value)
	}
	return sizeStr
}

//处理单文件传输
func doSingleFileHandler(conn net.Conn) {
	//响应一下
	//ynet.SendResponse(conn, ycomm.ResponseInfo{Ok: true, Message: "ok", Status: "ok"})
	logf("响应ok,准备接收单文件大小等信息>>>>")
	rMsg := ycomm.ReadMsg(conn)
	logf("收到单文件传输数据[", rMsg, "]>>>>>>")

	//数据格式 => 文件名称+大小
	fileDataMap := ycomm.ParseStrToMapData(rMsg)

	fileName := fileDataMap[ycomm.FILE_NAME]
	fileSizeStr := fileDataMap[ycomm.FILE_SIZE]
	logf("解析后文件名[", fileName, "] 文件大小[", fileSizeStr, "]B")

	//将string转换为64位int
	fileSize, _ := strconv.ParseInt(fileSizeStr, 10, 64)
	fmt.Println("正在接收文件【", fileName, "】 文件大小【", fileSizeReadable(fileSize), "】")

	//响应yroute可以发送数据了
	ynet.SendResponse(conn, ycomm.ResponseInfo{Ok: true, Message: "Prepare fileStream", Status: "Ok"})

	logf(">>>>准备接收文件流数据>>>>")
	recvSmallFile(conn, fileName, fileSize)
	logf(">>>>接收文件流数据完毕,结束>>>>")

}

func doHandlerRequest(conn net.Conn) bool {
	//请求命令数据
	rMsg := ycomm.ReadMsg(conn)
	fmt.Println("接收到Client：" + conn.RemoteAddr().String() + " 请求类型：" + rMsg)
	if rMsg == "d" {

		wg.Add(1)
		go doDirHandler(conn)
	} else if rMsg == "f" {

		wg.Add(1)
		go doFileHandler(conn)

	} else if rMsg == ycomm.YROUTE_CHECK_YRECV { //通用性检查命令
		logf("收到来自yrout的连接检查命令>>>>")

		ynet.SendResponse(conn, ycomm.ResponseInfo{Ok: true, Message: "ok", Status: "ok"})
		logf("响应ok信息给route>>>>")

	} else if rMsg == ycomm.YSEND_TO_YROUTE_TO_YRECV_SINGLE_FILE {
		logf("收到来自yroute的YROUTE_SEND_SINGLE_FILE命令")
		go doSingleFileHandler(conn)

	} else {
		fmt.Println("======非法数据传入========")
		return false
	}
	return true
}

func doAcceptMultiFileTranServer(netS net.Listener) {
	fmt.Println("MServer Successful running in ", netS.Addr().String())
	for {
		conn, err := netS.Accept()
		if err != nil {
			fmt.Println("listener.Accept() err1:", err)
			continue
		}

		if doHandlerRequest(conn) == false {
			logf("错误: Yrecv主程序结束")
			break
		}

	}
}

func doMultiFileTranServer() {
	//多文件传输建立port
	netS := doCreateServer(mutilFilePort)
	if netS != nil {
		defer netS.Close()
		doAcceptMultiFileTranServer(netS)
	}

}

//服务自动绑定
//
//含单文件传输服务和多文件传输服务
//
func doAutoBindServer() {
	logf("自动绑定模式===>开始创建单文件传输服务")
	singleServer := doCreateServer(singleFilePort)
	if singleServer != nil {
		defer singleServer.Close()
		fmt.Println("SServer Successful running in ", singleServer.Addr().String())
		wg.Add(1)
		go singleFileReceiver(singleServer)
	}

	//do udp fun
	wg.Add(1)
	go func() {
		defer wg.Done()
		listenUDP(ycomm.LOCAL_HOST)
		//if singleServer != nil {
		//	ipStr := singleServer.Addr().String()
		//	listenUDP(ipStr)
		//} else {
		//	listenUDP("nil")
		//}

	}()

	doMultiFileTranServer()
}

//根据指定ip建立8848,8849,8850服务
func doSpecificBindServer(ip string) {
	singleServer := ynet.GetBindNet(ip, singleFilePort)
	if singleServer != nil {
		defer singleServer.Close()
		fmt.Println("SServer Successful running in ", singleServer.Addr().String())
		wg.Add(1)
		go singleFileReceiver(singleServer)
	} else {
		fmt.Println("SServer 绑定失败")
	}

	//do udp fun
	wg.Add(1)
	go func() {
		defer wg.Done()
		listenUDP(ip)
	}()

	MServer := ynet.GetBindNet(ip, mutilFilePort)
	if MServer != nil {
		defer MServer.Close()
		doAcceptMultiFileTranServer(MServer)
	}

}

func doHelp() {
	fmt.Println("args must be => yrecv")
	fmt.Println("args must be => yrecv -b 本地监听ip")
}

//打印
func gLog(a ...interface{}) (n int, err error) {

	return fmt.Println(a)
}

var flags struct {
	ListenIP string
	RouteIP  string
}

func getUsage() {
	flag.Usage()
	fmt.Println("例如: yrecv")
	fmt.Println("例如: yrecv -b 10.3.4.2")
	fmt.Println("例如: yrecv -c 150.33.44.23 -b 10.3.4.2")
}

//监听BaseConn事件
func doListenBaseConnEvent(conn net.Conn) {

	logf(">>>>监听BaseConn启动")
	for {
		//收取信息
		rcvBytes, err0 := ycomm.ReadByte0(conn)
		if err0 != nil {
			fmt.Println("BaseConn 读取出错,退出")
			break
		}
		rcvMsg := string(rcvBytes)
		logf("收到Yroute BaseConn 信息>>>>", rcvMsg)
		reqInfo := ycomm.ParseStrToRequestInfo(rcvMsg)

		if reqInfo.Cmd == ycomm.YRECV_BASECONN_SINGLE { //大海模式 单文件传输
			logf("收到route大海模式传输单文件请求>>>>>>")
			yroutConn, err1 := ynet.GetRemoteConnection(flags.RouteIP, ycomm.RoutePort)
			if err1 != nil {
				logf("大海模式yrecv与yroute建立链接失败>>>>>>")
				//如果在于route建立连接的情况下出错,返回失败信息
				ynet.SendResponse(conn, ycomm.ResponseInfo{
					Ok:      false,
					Message: err1.Error(),
					Status:  "no",
				})
			} else {
				ynet.SendResponse(conn, ycomm.ResponseInfo{
					Ok:      true,
					Message: "ok",
					Status:  "ok",
				})
				logf("大海模式yrecv与yroute建立链接成功>>>>>>")
				logf("大海模式yrecv向yroute发送YRECV_REQUEST_ESTABLISH_CONN请求>>>>>>")

				var dMap = make(map[string]string)
				dMap[ycomm.TO_TYPTE] = ycomm.SINGLE
				dMap[ycomm.HOSTNAME] = ycomm.GetHostName()
				mapStr := ycomm.ParseMapToStr(dMap)
				logf("发送数据>>>", mapStr)

				ynet.SendRequest(yroutConn, ycomm.RequestInfo{
					Cmd:   ycomm.YRECV_REQUEST_ESTABLISH_CONN,
					Data:  mapStr,
					Other: "no",
				})

				//开始文件处理
				logf("大海模式: 准备接收文件数据>>>>>>")
				go doHandlerRequest(yroutConn)
				logf("大海模式: 接收文件数据>>>>>>结束")

			}

		} else if reqInfo.Cmd == ycomm.YRECV_BASECONN_HEADRTBEAT {
			//心跳
			logf("收到心跳>>>>>>")

		} else if reqInfo.Cmd == ycomm.YSEND_DIR_DATA_SYNC {
			logf("收到YSEND_DIR_DATA_SYNC>>>>开始处理")

			newYRoutConn, err01 := ynet.Socket(flags.RouteIP, ycomm.RoutePort)
			if err01 != nil {
				logf("创建同步目录数据连接失败>>>>>", err01)
				ynet.SendResponse(conn, ycomm.ResponseInfo{
					Ok:      false,
					Message: "创建同步目录数据连接失败>>>" + err01.Error(),
				})
			}

			var dMap = make(map[string]string)
			dMap[ycomm.TO_TYPTE] = ycomm.SINGLE
			dMap[ycomm.HOSTNAME] = ycomm.GetHostName()
			mapStr1 := ycomm.ParseMapToStr(dMap)
			logf("创建同步连接发送数据>>>", mapStr1)

			ynet.SendRequest(newYRoutConn, ycomm.RequestInfo{
				Cmd:   ycomm.YRECV_REQUEST_ESTABLISH_CONN,
				Data:  mapStr1,
				Other: "no",
			})

			//返回信息
			logf("Ok 开始同步目录数据...>>>>>>")
			ynet.SendResponse(conn, ycomm.ResponseInfo{
				Ok:      true,
				Message: "Ok 开始同步目录数据...",
			})

			logf("准备从Route读取目录数据...>>>>>>")
			rMsg := ycomm.ReadMsg(newYRoutConn)
			if rMsg == "d" {
				logf("收到YSEND_DIR_DATA_SYNC>>>>开始处理目录数据>>>>")
				doDirCreate(newYRoutConn)
				logf("处理目录数据完毕>>>>")
			}

		} else if reqInfo.Cmd == ycomm.YSEND_MUL_FILE_SYNC {
			logf("收到YSEND_MUL_FILE_SYNC>>>>开始处理>>>", reqInfo.ParseToJsonStr())
			dMap := ycomm.ParseStrToMapData(reqInfo.Data)
			coreStr := dMap[ycomm.CORE_NUM]

			coreNum, _ := strconv.Atoi(coreStr)

			var connList = make([]net.Conn, coreNum)
			var flag01 = false
			for i := 0; i < coreNum; i++ {
				routConn, err01 := ynet.Socket(flags.RouteIP, ycomm.RoutePort)
				if err01 != nil {
					flag01 = true
					break
				}
				connList[i] = routConn
			}
			if flag01 {
				ynet.SendResponse(conn, ycomm.ResponseInfo{
					Ok:      false,
					Message: "创建连接失败",
				})

				for i := 0; i < coreNum; i++ {
					//循环关闭
					connList[i].Close()
				}

				continue
			}
			//成功返回
			ynet.SendResponse(conn, ycomm.ResponseInfo{
				Ok:      true,
				Message: "创建连接成功",
			})
			logf("创建", coreNum, "个连接成功>>>>")

			for i := 0; i < coreNum; i++ {
				logf("准备向yrout发起YRECV_REQUEST_ESTABLISH_CONN=>MULTI>>>>>>>", i)
				go doParepareAcceptFileStream(connList[i])
			}

			logf("创建所有连接完毕>>>>")

		}

	}
}

func doParepareAcceptFileStream(conn net.Conn) {

	var dMap = make(map[string]string)
	dMap[ycomm.TO_TYPTE] = ycomm.MULTI
	dMap[ycomm.HOSTNAME] = ycomm.GetHostName()
	mapStr := ycomm.ParseMapToStr(dMap)
	logf("发送数据>>>", mapStr)

	ynet.SendRequest(conn, ycomm.RequestInfo{
		Cmd:   ycomm.YRECV_REQUEST_ESTABLISH_CONN,
		Data:  mapStr,
		Other: "no",
	})

	//开始文件处理
	logf("大海模式: 准备接收文件流数据>>>>>>")
	doHandlerRequest(conn)
	logf("大海模式: 接收文件流数据>>>>>>结束")

}

func doRouter(routeIp, listenIp string) {
	conn, err0 := ynet.GetRemoteConnection(routeIp, ycomm.RoutePort)
	if err0 != nil {
		logf(">>>连接远程Route[", routeIp+":"+ycomm.RoutePort, "]异常>>>>", err0)
		return
	}
	logf("yrecv成功连接远程[" + routeIp + ":" + ycomm.RoutePort + "]Router>>>>")

	hostname, _ := os.Hostname()
	coreN := runtime.NumCPU() * 2

	//数据载体
	yrbInfo := ycomm.YrecvBase{Cpu: strconv.Itoa(coreN), Name: hostname, Ip: listenIp}
	//请求信息
	req := ycomm.RequestInfo{
		Cmd:   ycomm.YRECV_INIT,
		Data:  ycomm.ParseYrecvBaseToJsonStr(yrbInfo),
		Other: "no",
	}

	//ycomm.WriteMsg(conn, req.ParseToJsonStr())
	ynet.SendRequest(conn, req)
	logf(">>>发送注册信息>>>", req.ParseToJsonStr(), ">>>到Route")

	msgStr := ycomm.ReadMsg(conn)
	logf(">>>收到Route响应数据>>>", msgStr)

	//
	go doListenBaseConnEvent(conn)

	logf(">>>>>启动单文件,多文件传输及探测响应服务")
	doSpecificBindServer(listenIp)
	//阻塞
	<-exitFlag

}

var exitFlag = make(chan bool)

func main() {

	flag.StringVar(&flags.RouteIP, "c", "", "路由ip")
	flag.StringVar(&flags.ListenIP, "b", "", "指定监听ip")
	flag.BoolVar(&ycomm.Debug, "debug", false, "debug mode")
	flag.Parse()

	logf("输入参数==>[", flags, "]")

	argLen := len(os.Args) //获取参数长度

	if argLen == 1 { //默认自动获取ip监听模式
		logf("自动获取ip模式=====>>>>>>")
		doAutoBindServer()
		wg.Wait()

	} else if flags.RouteIP != "" { //如果选择路由模式
		//暂时路由模式 必须指定本地绑定ip
		if flags.ListenIP == "" {
			fmt.Println("错误(ERROR): 必须指定ListenIP")
			os.Exit(01)
		}
		logf("路由模式=====>>>>>>路由ip[", flags.RouteIP, "]===>本地绑定ip[", flags.ListenIP, "]")

		doRouter(flags.RouteIP, flags.ListenIP)

	} else if flags.ListenIP != "" { //如果是指定ip模式
		logf("指定ip模式=====>>>>>>[", flags.ListenIP, "]")
		doSpecificBindServer(flags.ListenIP)
		wg.Wait()
	} else {
		getUsage()
	}

}
