package ycomm

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

//版本号
const YROUTER_V = "20220603"
const YRECV_V = "20220603"
const YSEND_V = "20220603"

//Yrecv BaseConn指令

//大海模式(单文件) route->yrecv : 表示要求yrecv端主要要与yroute建立连接
const YRECV_BASECONN_SINGLE = "ybcs"

//(心跳) route->yrecv
const YRECV_BASECONN_HEADRTBEAT = "ybcht"

//9949请求指令
//yrout向yrecv发送检查连通性命令
const YROUTE_CHECK_YRECV = "check"

const YSEND_DIR_DATA_SYNC = "ydds"

const YSEND_MUL_FILE_SYNC = "ymfs"

const YSEND_MUL_FILE_SYNC2 = "ymfs2"

//route 服务命令
const (
	YRECV_INIT                   = "yrecv_init" //yrecv注册信息
	YDECT_MSG                    = "ydect_msg"  //ydect探测信息
	YRECV_REQUEST_ESTABLISH_CONN = "yreconn"    //yrecv主动请求建立连接
	//ysend向yrecv发送单个文件
	YSEND_TO_YROUTE_TO_YRECV_SINGLE_FILE = "y_s_s_f"
	//ysend=>yrecv多文件发送
	YSEND_TO_YROUTE_TO_YRECV_MULTI_FILE = "yssmf"
)

const (
	SizeB          int64 = 1024
	SizeKB         int64 = 1048576
	SizeMB         int64 = 1073741824
	SizeGB         int64 = 1099511627776
	B                    = 1
	KB                   = 2
	MB                   = 3
	GB                   = 4
	GOBAL_TASK_NUM       = 3
	Size1B               = 1
	Size1KB              = Size1B * 1024
	Size1MB              = Size1KB * 1024
	Size1GB              = Size1MB * 1024
)

const TO_TYPTE = "to_type"
const SINGLE = "Single"
const MULTI = "Multi"
const HOSTNAME = "hostname"

//一般代码yrecv注册名称
const REMOTE_NAME = "remoteName"

const (
	MultiRemotePort  = "9949"
	SingleRemotePort = "8848" //单文件端口
	RoutePort        = "9950" //route端口
	LOCAL_HOST       = "0.0.0.0"
)

//字段常量名称
const FILE_NAME = "fileName"
const FILE_SIZE = "fileSize"
const SEND_TO_NAME = "name"
const CORE_NUM = "coreNum"

//注册信息 RequestInfo{cmd: "yrecv_init", data: "name:yms ip:192.168.25.88 cpu:8", other:""}
//	响应信息ResponseInfo{ok: true, message:"Recvive", status:"OK"}

//ydect探测信息 RequestInfo{cmd: "ydect_msg", data:"msg:no",other:""}
//	响应信息ResponseInfo{ok: true, message:"[]

var Debug bool

//yrecv注册基本信息
type YrecvBase struct {
	Name string `json:"name"` //yrecv端机器简称,用于映射key
	Ip   string `json:"ip"`   //yrecv端ip
	Cpu  string `json:"cpu"`  //yrecv端口cpu核心数
}

func (yrb *YrecvBase) Show() {
	fmt.Printf("[名字: %s, ip: %s, 核心数: %s]\n", yrb.Name, yrb.Ip, yrb.Cpu)
}

func (yrb *YrecvBase) ParseToJsonStr() string {
	bytes, _ := json.Marshal(yrb)
	jStr := string(bytes)

	return jStr
}

//list-> string
func ParseYrecvBaseToJsonStr(yrb YrecvBase) string {
	var yrecvList = make([]YrecvBase, 1)
	yrecvList[0] = yrb

	return ParseYrecvBaseListToJsonStr(yrecvList)
}

//list-> string
func ParseYrecvBaseListToJsonStr(yrecvList []YrecvBase) string {
	bytes, _ := json.Marshal(&yrecvList)
	jStr := string(bytes)

	return jStr
}

//jstr -> single
func ParseStrToYrecvBase(jsonStr string) YrecvBase {
	list := ParseStrToYrecvBaseList(jsonStr)
	if len(list) == 0 {
		fmt.Println("错误(ERROR): ParseStrToYrecvBase=>", jsonStr, ">>>数据长度为0")

	}
	return list[0]
}

//str->list
func ParseStrToYrecvBaseList(jsonStr string) []YrecvBase {
	var bytes = []byte(jsonStr)

	newList := []YrecvBase{}
	json.Unmarshal(bytes, &newList)

	return newList
}

type CFileInfo struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
	Path  string `json:"path"`
}

type RequestInfo struct {
	Cmd   string `json:"cmd"`
	Data  string `json:"data"`
	Other string `json:"other"`
}

//Req -> byte
func (req RequestInfo) ParseToByte() []byte {
	bytes, _ := json.Marshal(req)
	return bytes
}

//req.data => map
func (req RequestInfo) GetDataMap() map[string]string {
	str := req.Data

	var resMap = make(map[string]string)
	split1 := strings.Split(str, " ")
	for _, para := range split1 {
		kv := strings.Split(para, ":")
		resMap[kv[0]] = kv[1]
	}

	return resMap
}

//Req->Str
func (req RequestInfo) ParseToJsonStr() string {
	return string(req.ParseToByte())
}

// byte -> Req
func ParseByteToRequestInfo(bytes []byte) RequestInfo {
	var result RequestInfo

	json.Unmarshal(bytes, &result)
	return result
}

// str->Req
func ParseStrToRequestInfo(str string) RequestInfo {
	return ParseByteToRequestInfo([]byte(str))
}

type ResponseInfo struct {
	Ok      bool   `json:"ok"`
	Message string `json:"message"`

	//用于自定义通信规则时使用 其他不用是 传 OK
	Status string `json:"status"`
}

func ParseResponseToBytes(res ResponseInfo) []byte {
	bytes, _ := json.Marshal(res)
	return bytes
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

//发送数据
func WriteMsg(conn net.Conn, msg string) bool {
	sendMsg := []byte(msg + msgCloseFlag) // 13
	_, err := conn.Write(sendMsg)
	if err != nil {
		fmt.Printf("writeMsg【%s】失败\n", sendMsg)
		fmt.Println("错误(ERROR): WriteMsg发送数据出错[", err, "]")
		//panic(err)
		return false
	}
	//fmt.Println("发送数据【"+msg+"】长度=", wLen)
	return true
}

func ReadByte0(conn net.Conn) (rcvBytes []byte, err error) {

	tmpByte := make([]byte, 1)
	var cnt int = 0

	for {
		_, err := conn.Read(tmpByte)
		if err != nil {
			fmt.Println("错误(ERROR): ReadByte读取数据出错[", err, "]")
			//panic(err)
			err = errors.New("异常：" + err.Error())
			return nil, err
		}
		if tmpByte[0] == byte(13) {
			break
		}

		rcvBytes = append(rcvBytes, tmpByte[0])
		cnt++
	}

	return rcvBytes, nil
}

func ReadByte(conn net.Conn) []byte {
	byte0, _ := ReadByte0(conn)

	return byte0
}

//以 \r 结尾
func ReadMsg(conn net.Conn) string {
	return string(ReadByte(conn))
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

// GetLocalIpv4List 获取所有ipv4绑定列表
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

// ParseUdpFormat 解析ipv4为数组格式
func ParseUdpFormat(ip string) []byte {
	res := make([]byte, 4)
	//splitStr := strings.Split(ip,":")
	//ipStr := splitStr[0]
	//port  := splitStr[1]
	//intPort, _ := strconv.Atoi(port)

	p := strings.Split(ip, ".")
	a, _ := strconv.Atoi(p[0])
	b, _ := strconv.Atoi(p[1])
	c, _ := strconv.Atoi(p[2])
	d, _ := strconv.Atoi(p[3])

	res[0] = byte(a)
	res[1] = byte(b)
	res[2] = byte(c)
	res[3] = byte(d)

	return res
}

// ParseStrToMapData 以/n作为数据分隔,以/a作为kv分隔
//解析示例：str := "name\ayms\nfileName\ahi.txt\nfileSize\a2342";
//	fileSize => 2342
//	name => yms
//	fileName => hi.txt
func ParseStrToMapData(str string) map[string]string {
	resMap := make(map[string]string)
	sList := strings.Split(str, "\n")
	for _, v := range sList {
		s2 := strings.Split(v, "\a")
		resMap[s2[0]] = s2[1]
	}

	return resMap
}

func ParseMapToStr(dMap map[string]string) string {
	var reStr = ""
	n := len(dMap)
	i := 0
	for k, v := range dMap {
		if i+1 == n {
			reStr += k + "\a" + v
		} else {
			reStr += k + "\a" + v + "\n"
		}
		i++
	}
	return reStr
}

func GetHostName() string {
	hostname, _ := os.Hostname()
	return hostname
}

type FileSliceInfo struct {
	Id     int    `json:"id"`
	Name   string `json:"name"`
	Start  int64  `json:"start"`
	Length int64  `json:"length"`
	Hash   string `json:"hash"`
}

func (fsi *FileSliceInfo) FileSliceInfoShow() {
	fmt.Println("id: ", fsi.Id, " Name: ", fsi.Name, " Start: ", fsi.Start, " Length: ", fsi.Length, " Hash: ", fsi.Hash)
}

func (fsi *FileSliceInfo) ParseFileSliceToJsonStr() string {
	bytes, _ := json.Marshal(fsi)
	jStr := string(bytes)
	return jStr
}

func ParseByteToFileSlice(bytes []byte) FileSliceInfo {
	fsi := FileSliceInfo{}
	json.Unmarshal(bytes, &fsi)
	return fsi
}

func ParseStrToFileSlice(jsonStr string) FileSliceInfo {
	var bytes = []byte(jsonStr)
	fsi := FileSliceInfo{}
	json.Unmarshal(bytes, &fsi)
	return fsi
}

//分片文件传输指令
const (
	FILE_INFO_TIP             = "file_info_tip"
	SMALL_FILE                = "small_file"                //小文件传输指令
	BIGFILE_INIT              = "bigfile_init"              //大文件分片初始命令
	BIGFILE_SLICE_SYNC        = "bigfile_slice_sync"        //同步命令
	BIGFILE_SLICE_SYNC_FINISH = "bigfile_slice_sync_finish" //同步命令完成
)

const SPLIT_FLAG = "___"
const TmpSlice = "tmpSlice"

func IsSmallSend(sendFileSize int64) bool {
	if sendFileSize <= 1*SizeGB {
		return true
	}
	return false
}

//
func Md5Hash(input []byte) string {
	c := md5.New()
	c.Write(input)
	bytes := c.Sum(nil)
	return hex.EncodeToString(bytes)
}

//get Os Sparator
func GetOsSparator() string {
	const separator = os.PathSeparator
	if separator == '\\' {
		return "\\"
	} else {
		return "/"
	}
}

//tmp\shadow\test2.go => test2.go
func GetFileName(fName string) string {
	split := strings.Split(fName, GetOsSparator())
	return split[len(split)-1]
}

func FileReadByOffset(file *os.File, startOffset int64, length int64) ([]byte, error) {
	// 创建一个字节切片来存储读取到的数据
	data := make([]byte, length)

	// 移动到起始位置
	_, err := file.Seek(startOffset, io.SeekStart)
	if err != nil {
		fmt.Println("移动文件指针失败:", err)
		return nil, err
	}

	// 读取数据
	_, err = file.Read(data)
	if err != nil && err != io.EOF {
		fmt.Println("读取数据失败:", err)
		return nil, err
	}

	return data, nil
}

func FileReadByOffset1(file *os.File, data []byte, startOffset int64) error {
	// 移动到起始位置
	_, err := file.Seek(startOffset, io.SeekStart)
	if err != nil {
		fmt.Println("移动文件指针失败:", err)
		return err
	}

	// 读取数据
	_, err = file.Read(data)
	if err != nil && err != io.EOF {
		fmt.Println("读取数据失败:", err)
		return err
	}

	return nil
}
