package main

import (
	"flag"
	"runtime"
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

func main() {

	ysendUsage()

	//flag.PrintDefaults()

}
