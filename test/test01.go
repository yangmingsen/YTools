package main

import (
	"YTools/ycomm"
	"encoding/json"
	"flag"
	"fmt"
	"runtime"
	"strconv"
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

func main() {
	parseArr()
}

// 结构体定义
type robot struct {
	Name   string `json:"name"`
	Amount int    `json:"amount"`
}

// 解析到结构体数组
func parse_array() {
	fmt.Println("解析json字符串为结构体数组")
	str := "[{\"name\":\"name1\",\"amount\":100},{\"name\":\"name2\",\"amount\":200},{\"name\":\"name3\",\"amount\":300},{\"name\":\"name4\",\"amount\":400}]"
	all := []robot{}
	err := json.Unmarshal([]byte(str), &all)
	if err != nil {
		fmt.Printf("err=%v", err)
	}
	for _, one := range all {
		fmt.Printf("name=%v, amount=%v\n", one.Name, one.Amount)
	}
}

// 解析到结构体指针的数组
func parse_pointer_array() {
	fmt.Println("解析json字符串为结构体指针的数组")
	str := "[{\"name\":\"name1\",\"amount\":100},{\"name\":\"name2\",\"amount\":200},{\"name\":\"name3\",\"amount\":300},{\"name\":\"name4\",\"amount\":400}]"
	all := []*robot{}
	err := json.Unmarshal([]byte(str), &all)
	if err != nil {
		fmt.Printf("err=%v", err)
	}
	for _, one := range all {
		fmt.Printf("name=%v, amount=%v\n", one.Name, one.Amount)
	}

}
