package main

import (
	"YTools/ycomm"
	"encoding/json"
	"flag"
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

func ysendUsage() {
	var flags struct {
		RouteIP  string
		RemoteIP string
		FilePath string
		goNumber int
	}
	flag.StringVar(&flags.RouteIP, "c", "", "路由ip")
	flag.StringVar(&flags.FilePath, "r", "", "./文件夹")
	flag.StringVar(&flags.FilePath, "f", "", "./文件")
	flag.StringVar(&flags.RemoteIP, "d", "", "目标ip")
	flag.IntVar(&flags.goNumber, "sn", runtime.NumCPU()*2, "并发数[默认cpu*2]")
	flag.Parse()

	flag.Usage()
}

var mMap = make(map[string]string)

func hello() {
	req := ycomm.RequestInfo{Cmd: "yrecv_ini", Data: "name:yms cpu:16 ip:10.3.4.3", Other: ""}

	dataMap := req.GetDataMap()

	var yrecvhh = YrecvRegInfo{}
	yrecvhh.parseMap(dataMap)
	fmt.Println(yrecvhh)
	fmt.Println("------------")
	for k, v := range dataMap {
		fmt.Println("k=", k, " v=", v)
	}

	fmt.Println(req.ParseToByte())
	fmt.Println("jsonStr=", req.ParseToJsonStr())
}

type YrecvRegInfo struct {
	Name string //yrecv端机器简称,用于映射key
	Ip   string //yrecv端ip
	Cpu  string //yrecv端口cpu核心数
}

func (yrb *YrecvRegInfo) Show() {
	fmt.Printf("[名字:%s, ip:%s, 核心数:%s]\n", yrb.Name, yrb.Ip, yrb.Cpu)
}

func (yrb *YrecvRegInfo) ParseToJsonStr() string {
	bytes, _ := json.Marshal(yrb)
	jStr := string(bytes)

	return jStr
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
	fmt.Println("yrr=", yrr)

}

func parseArr() {
	var yrecvList = make([]YrecvRegInfo, 3)
	for i := 0; i < 3; i++ {
		yrecvList[i].Name = "yms" + strconv.Itoa(int(i))
		yrecvList[i].Cpu = "cpu" + strconv.Itoa(int(i))
		yrecvList[i].Ip = "10.2.3." + strconv.Itoa(int(i))
	}

	fmt.Println(yrecvList[0].ParseToJsonStr())

	//list -> str
	bytes, _ := json.Marshal(&yrecvList)
	jStr := string(bytes)
	fmt.Println("jStr=", jStr)

	//byte -> list
	newList := []YrecvRegInfo{}
	json.Unmarshal(bytes, &newList)
	for _, y := range newList {
		//fmt.Println("x=",x,"  y=",y)
		y.Show()
	}

}

//解析示例：str := "name\ayms\nfileName\ahi.txt\nfileSize\a2342";
//	fileSize => 2342
//	name => yms
//	fileName => hi.txt
func ParseStrToDataMap(str string) map[string]string {
	resMap := make(map[string]string)
	sList := strings.Split(str, "\n")
	for _, v := range sList {
		s2 := strings.Split(v, "\a")
		resMap[s2[0]] = s2[1]
	}

	return resMap
}

func main() {
	var dMap = make(map[string]string)
	dMap["hi"] = "yms"
	dMap["nice"] = "10"
	dMap["what"] = "okok"
	dMap["when"] = "20201012"
	str := ycomm.ParseMapToStr(dMap)

	fmt.Print("str=", str)
	fmt.Print("--------")

}

func main12() {
	str := "name\ayms\nfileName\ahi.txt\nfileSize\a2342"
	dataMap := ParseStrToDataMap(str)
	toStr := ycomm.ParseMapToStr(dataMap)

	fmt.Print(toStr)
	fmt.Println("-----------")

}

func main1() {
	xx := Robot{Name: "yangmingsen", Amount: 10}
	fmt.Println("xx=>", xx)
	updateT(&xx)
	fmt.Println("xx=>", xx)

}

func updateT(rb *Robot) {
	rb.Name = "yms"
	rb.Amount = 23
}

// 结构体定义
type Robot struct {
	Name   string `json:"name"`
	Amount int    `json:"amount"`
}
